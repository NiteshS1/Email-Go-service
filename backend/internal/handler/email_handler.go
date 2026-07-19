package handler

import (
	"fmt"
	"log/slog"

	"github.com/emailservice/internal/service"
	fiber "github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const traceIDHeader = "X-Trace-ID"

// EmailHandler handles HTTP requests for email operations.
type EmailHandler struct {
	service    service.EmailService
	newService service.EmailService
}

func NewEmailHandler(service service.EmailService, newService service.EmailService) *EmailHandler {
	return &EmailHandler{service, newService}
}

// SendEmail handles POST /emails/send requests.
func (h *EmailHandler) SendEmail(c *fiber.Ctx) error {
	var req service.SendEmailRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Resolve trace ID: body > header > generated
	if req.TraceID == "" {
		req.TraceID = c.Get(traceIDHeader)
	}
	if req.TraceID == "" {
		req.TraceID = uuid.New().String()
	}

	if req.Receiver == "" || req.Template == "" || req.Subject == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "receiver_email, template, and subject are required",
		})
	}

	// Pass Fiber's user context for tracing propagation
	if err := h.service.SendEmail(c.UserContext(), req); err != nil {
		slog.Error("send email failed", "trace_id", req.TraceID, "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process email request",
		})
	}

	c.Set(traceIDHeader, req.TraceID)
	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"message":  "Email processed successfully",
		"trace_id": req.TraceID,
	})
}

func (h *EmailHandler) SendEmailHandler(c *fiber.Ctx) error {
	var req service.EmailRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request format"})
	}

	if req.Recipient == "" || req.Subject == "" || req.Body == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Missing required fields"})
	}

	err := h.newService.SendOTPEmailRequest(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("Failed to send email: %v", err)})
	}

	return c.JSON(fiber.Map{"message": "Email sent successfully"})
}
