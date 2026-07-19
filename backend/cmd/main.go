package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/emailservice/internal/config"
	"github.com/emailservice/internal/consumer"
	"github.com/emailservice/internal/fetcher"
	"github.com/emailservice/internal/handler"
	"github.com/emailservice/internal/infrastructure"
	"github.com/emailservice/internal/repository"
	"github.com/emailservice/internal/routes"
	"github.com/emailservice/internal/service"
	otel_middleware "github.com/gofiber/contrib/otelfiber/v2"
	fiber "github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	// ─── Structured Logging ────────────────────────────────────────────────
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// ─── Configuration ─────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration error", "error", err)
		os.Exit(1)
	}

	// ─── OpenTelemetry Tracing ─────────────────────────────────────────────
	tp := initTracer(cfg.OTELEndpoint)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(shutdownCtx); err != nil {
			slog.Error("tracer shutdown error", "error", err)
		}
	}()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// ─── Database & Migrations ─────────────────────────────────────────────
	if err := infrastructure.RunMigrations(cfg); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}

	db, err := infrastructure.NewDB(cfg)
	if err != nil {
		slog.Error("database init failed", "error", err)
		os.Exit(1)
	}

	// Ensure DB connections are returned to pool on shutdown
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	// ─── Dependency Wiring ─────────────────────────────────────────────────
	repo := repository.NewEmailRepository(db)
	sender := infrastructure.NewSMTPSender(cfg)
	svc := service.NewEmailService(repo, fetcher.NewCombinedFetcher(), sender)
	h := handler.NewEmailHandler(svc, svc)

	// ─── Context for graceful shutdown ─────────────────────────────────────
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ─── RabbitMQ Consumer ─────────────────────────────────────────────────
	go consumer.Run(ctx, cfg.RabbitMQURL, svc)

	// ─── HTTP Server ───────────────────────────────────────────────────────
	app := fiber.New(fiber.Config{
		// Don't expose internal error details in 500 responses
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			slog.Error("unhandled request error", "path", c.Path(), "error", err)
			return c.Status(code).JSON(fiber.Map{"error": "Internal server error"})
		},
	})

	app.Use(recover.New()) // Recover from panics without crashing
	app.Use(otel_middleware.Middleware())
	routes.Email(app, h)

	// Health check endpoint (used by load balancers / k8s probes)
	app.Get("/health", func(c *fiber.Ctx) error {
		sqlDB, err := db.DB()
		if err != nil || sqlDB.Ping() != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "unhealthy", "reason": "database"})
		}
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Start server in a goroutine so we can wait for shutdown
	go func() {
		addr := ":" + cfg.AppPort
		slog.Info("HTTP server starting", "addr", addr)
		if err := app.Listen(addr); err != nil {
			slog.Error("HTTP server error", "error", err)
		}
	}()

	// Block until a signal is received
	<-ctx.Done()
	slog.Info("shutdown signal received, draining...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		slog.Error("HTTP server shutdown error", "error", err)
	}

	slog.Info("service stopped cleanly")
}

// initTracer sets up an OTLP trace exporter when an endpoint is configured,
// falling back to a no-op provider so the service runs without a collector.
func initTracer(endpoint string) *sdktrace.TracerProvider {
	if endpoint == "" {
		slog.Info("OTEL_EXPORTER_OTLP_ENDPOINT not set; tracing disabled")
		return sdktrace.NewTracerProvider()
	}

	exp, err := otlptracehttp.New(
		context.Background(),
		otlptracehttp.WithEndpointURL(endpoint),
	)
	if err != nil {
		slog.Warn("OTLP exporter setup failed; running without export", "error", err)
		return sdktrace.NewTracerProvider()
	}

	slog.Info("OTLP trace exporter configured", "endpoint", endpoint)
	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
}
