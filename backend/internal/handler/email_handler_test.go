package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/emailservice/internal/handler"
	"github.com/emailservice/internal/service"
	fiber "github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Mock ────────────────────────────────────────────────────────────────────

type mockEmailService struct {
	fn func(ctx context.Context, req service.SendEmailRequest) error
}

func (m *mockEmailService) SendEmail(ctx context.Context, req service.SendEmailRequest) error {
	if m.fn != nil {
		return m.fn(ctx, req)
	}
	return nil
}

func (m *mockEmailService) SendOTPEmailRequest(req service.EmailRequest) error {
	return nil
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func newApp(svc service.EmailService) *fiber.App {
	app := fiber.New()
	h := handler.NewEmailHandler(svc, svc)
	app.Post("/emails/send", h.SendEmail)
	return app
}

func doPost(app *fiber.App, body interface{}, extraHeaders map[string]string) *httptest.ResponseRecorder {
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/emails/send", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}
	resp, _ := app.Test(req, -1)
	return &httptest.ResponseRecorder{
		Code: resp.StatusCode,
		Body: func() *bytes.Buffer {
			b, _ := io.ReadAll(resp.Body)
			return bytes.NewBuffer(b)
		}(),
	}
}

// ── Tests ────────────────────────────────────────────────────────────────────

func TestSendEmail_ValidRequest_Returns202(t *testing.T) {
	svc := &mockEmailService{}
	app := newApp(svc)

	rec := doPost(app, map[string]interface{}{
		"receiver_email": "alice@example.com",
		"template":       "welcome",
		"subject":        "Welcome!",
	}, nil)

	assert.Equal(t, fiber.StatusAccepted, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "Email processed successfully", resp["message"])
	assert.NotEmpty(t, resp["trace_id"])
}

func TestSendEmail_TraceIDFromHeader_UsedWhenBodyMissing(t *testing.T) {
	var capturedTraceID string
	svc := &mockEmailService{
		fn: func(_ context.Context, req service.SendEmailRequest) error {
			capturedTraceID = req.TraceID
			return nil
		},
	}
	app := newApp(svc)

	rec := doPost(app, map[string]interface{}{
		"receiver_email": "bob@example.com",
		"template":       "info",
		"subject":        "Info",
	}, map[string]string{"X-Trace-ID": "my-custom-trace"})

	assert.Equal(t, fiber.StatusAccepted, rec.Code)
	assert.Equal(t, "my-custom-trace", capturedTraceID)
}

func TestSendEmail_MissingReceiver_Returns400(t *testing.T) {
	svc := &mockEmailService{}
	app := newApp(svc)

	rec := doPost(app, map[string]interface{}{
		"template": "welcome",
		"subject":  "Hello",
		// receiver_email intentionally missing
	}, nil)

	assert.Equal(t, fiber.StatusBadRequest, rec.Code)
	var resp map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Contains(t, resp["error"], "required")
}

func TestSendEmail_MissingTemplate_Returns400(t *testing.T) {
	svc := &mockEmailService{}
	app := newApp(svc)

	rec := doPost(app, map[string]interface{}{
		"receiver_email": "alice@example.com",
		"subject":        "Hello",
		// template missing
	}, nil)

	assert.Equal(t, fiber.StatusBadRequest, rec.Code)
}

func TestSendEmail_ServiceError_Returns500WithGenericMessage(t *testing.T) {
	svc := &mockEmailService{
		fn: func(_ context.Context, _ service.SendEmailRequest) error {
			return errors.New("internal database exploded")
		},
	}
	app := newApp(svc)

	rec := doPost(app, map[string]interface{}{
		"receiver_email": "carol@example.com",
		"template":       "alert",
		"subject":        "Alert",
	}, nil)

	assert.Equal(t, fiber.StatusInternalServerError, rec.Code)

	var resp map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	// Internal details must NOT be leaked to the client
	assert.NotContains(t, resp["error"], "database exploded")
	assert.Equal(t, "Failed to process email request", resp["error"])
}

func TestSendEmail_InvalidJSON_Returns400(t *testing.T) {
	svc := &mockEmailService{}
	app := fiber.New()
	h := handler.NewEmailHandler(svc, svc)
	app.Post("/emails/send", h.SendEmail)

	req := httptest.NewRequest("POST", "/emails/send", bytes.NewReader([]byte("not-json{")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}
