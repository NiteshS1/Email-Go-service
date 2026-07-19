package consumer

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/emailservice/internal/service"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	queueName = "email.send"
	dlqName   = "email.send.dlq"
	prefetch  = 1
)

// Run starts the RabbitMQ consumer loop. It reconnects on failures and
// terminates cleanly when ctx is cancelled.
func Run(ctx context.Context, url string, svc service.EmailService) {
	if url == "" {
		slog.Info("RABBITMQ_URL not set; async consumer disabled")
		return
	}

	for {
		select {
		case <-ctx.Done():
			slog.Info("consumer: context cancelled, exiting")
			return
		default:
		}

		if err := runSession(ctx, url, svc); err != nil {
			slog.Error("RabbitMQ session error", "error", err)
		}

		select {
		case <-ctx.Done():
			slog.Info("consumer: context cancelled, exiting")
			return
		case <-time.After(5 * time.Second):
			slog.Info("consumer: reconnecting to RabbitMQ")
		}
	}
}

// runSession establishes one RabbitMQ connection+channel session and processes
// deliveries until the connection drops or ctx is cancelled.
func runSession(ctx context.Context, url string, svc service.EmailService) error {
	conn, err := amqp.Dial(url)
	if err != nil {
		return err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	if _, err = ch.QueueDeclare(queueName, true, false, false, false, nil); err != nil {
		return err
	}
	_, _ = ch.QueueDeclare(dlqName, true, false, false, false, nil)

	if err = ch.Qos(prefetch, 0, false); err != nil {
		return err
	}

	deliveries, err := ch.Consume(queueName, "email-service", false, false, false, false, nil)
	if err != nil {
		return err
	}

	slog.Info("RabbitMQ consumer ready", "queue", queueName)

	connClosed := conn.NotifyClose(make(chan *amqp.Error, 1))
	for {
		select {
		case <-ctx.Done():
			return nil
		case amqpErr := <-connClosed:
			if amqpErr != nil {
				return amqpErr
			}
			return nil
		case d, ok := <-deliveries:
			if !ok {
				return nil
			}
			handleDelivery(ctx, ch, d, svc)
		}
	}
}

const traceIDHeaderKey = "x-trace-id"

func handleDelivery(ctx context.Context, ch *amqp.Channel, d amqp.Delivery, svc service.EmailService) {
	var req service.SendEmailRequest
	if err := json.Unmarshal(d.Body, &req); err != nil {
		slog.Error("invalid message body", "error", err)
		_ = d.Nack(false, false) // dead-letter
		return
	}

	if req.TraceID == "" && d.Headers != nil {
		if v, ok := d.Headers[traceIDHeaderKey]; ok {
			if s, ok := v.(string); ok {
				req.TraceID = s
			}
		}
	}

	if req.TraceID == "" || req.Receiver == "" || req.Template == "" || req.Subject == "" {
		slog.Warn("missing required fields in message", "trace_id", req.TraceID)
		_ = d.Nack(false, false) // dead-letter
		return
	}

	if err := svc.SendEmail(ctx, req); err != nil {
		slog.Error("SendEmail failed", "trace_id", req.TraceID, "error", err)
		_ = d.Nack(false, true) // requeue
		return
	}

	_ = d.Ack(false)
}
