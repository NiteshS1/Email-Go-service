package service_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/emailservice/internal/domain"
	"github.com/emailservice/internal/infrastructure"
	"github.com/emailservice/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Mocks ────────────────────────────────────────────────────────────────────

type mockRepo struct {
	saved      *domain.Email
	findResult *domain.Email
	findErr    error
	saveErr    error
	updateErr  error
	updateCalls []struct {
		id     int64
		status domain.StatusType
		errMsg string
	}
}

func (m *mockRepo) Save(email *domain.Email) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	email.ID = 1
	m.saved = email
	return nil
}

func (m *mockRepo) FindByTraceID(_ string) (*domain.Email, error) {
	return m.findResult, m.findErr
}

func (m *mockRepo) UpdateStatus(id int64, status domain.StatusType, errMsg string) error {
	m.updateCalls = append(m.updateCalls, struct {
		id     int64
		status domain.StatusType
		errMsg string
	}{id, status, errMsg})
	return m.updateErr
}

type mockSender struct {
	sendFn func(to, subject, body string, attachments []infrastructure.Attachment) error
}

func (m *mockSender) Send(to, subject, body string, attachments []infrastructure.Attachment) error {
	if m.sendFn != nil {
		return m.sendFn(to, subject, body, attachments)
	}
	return nil
}

type mockFetcher struct {
	fetchFn func(url, name string) (string, func(), error)
}

func (m *mockFetcher) Fetch(url, name string) (string, func(), error) {
	if m.fetchFn != nil {
		return m.fetchFn(url, name)
	}
	return "", func() {}, nil
}

// ── Test helpers ─────────────────────────────────────────────────────────────

// writeTemplateFile creates a temporary template file and returns a cleanup func.
func writeTemplateFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdirall: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("write template: %v", err)
	}
}

func newRequest() service.SendEmailRequest {
	return service.SendEmailRequest{
		TraceID:   "trace-001",
		TenantID:  1,
		ServiceID: 2,
		Receiver:  "user@example.com",
		Template:  "test",
		Subject:   "Hello",
		Data:      map[string]interface{}{"Name": "World"},
	}
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestSendEmail_Success(t *testing.T) {
	// Arrange: create a real template file under the default templates path
	writeTemplateFile(t, "templates", "test.html", `Hello, {{.Name}}!`)
	defer os.RemoveAll("templates")

	repo := &mockRepo{}
	sender := &mockSender{}
	svc := service.NewEmailService(repo, &mockFetcher{}, sender)

	// Act
	err := svc.SendEmail(context.Background(), newRequest())

	// Assert
	require.NoError(t, err)
	require.Len(t, repo.updateCalls, 1)
	assert.Equal(t, domain.StatusSuccess, repo.updateCalls[0].status)
	assert.Empty(t, repo.updateCalls[0].errMsg)
}

func TestSendEmail_SMTPFailure_RecordsFailStatus(t *testing.T) {
	writeTemplateFile(t, "templates", "test.html", `Hello!`)
	defer os.RemoveAll("templates")

	repo := &mockRepo{}
	sender := &mockSender{
		sendFn: func(_, _, _ string, _ []infrastructure.Attachment) error {
			return errors.New("smtp timeout")
		},
	}
	svc := service.NewEmailService(repo, &mockFetcher{}, sender)

	err := svc.SendEmail(context.Background(), newRequest())

	require.NoError(t, err) // service returns nil; the failure is recorded in DB
	require.Len(t, repo.updateCalls, 1)
	assert.Equal(t, domain.StatusFail, repo.updateCalls[0].status)
	assert.Contains(t, repo.updateCalls[0].errMsg, "smtp timeout")
}

func TestSendEmail_TemplateNotFound_ReturnsError(t *testing.T) {
	repo := &mockRepo{}
	sender := &mockSender{}
	svc := service.NewEmailService(repo, &mockFetcher{}, sender)

	req := newRequest()
	req.Template = "nonexistent"

	err := svc.SendEmail(context.Background(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse template")
}

func TestSendEmail_RepositoryFindError_ReturnsError(t *testing.T) {
	writeTemplateFile(t, "templates", "test.html", `Hi!`)
	defer os.RemoveAll("templates")

	repo := &mockRepo{findErr: errors.New("db connection lost")}
	sender := &mockSender{}
	svc := service.NewEmailService(repo, &mockFetcher{}, sender)

	err := svc.SendEmail(context.Background(), newRequest())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db connection lost")
}

func TestSendEmail_RepositorySaveError_ReturnsError(t *testing.T) {
	writeTemplateFile(t, "templates", "test.html", `Hi!`)
	defer os.RemoveAll("templates")

	repo := &mockRepo{saveErr: errors.New("insert failed")}
	sender := &mockSender{}
	svc := service.NewEmailService(repo, &mockFetcher{}, sender)

	err := svc.SendEmail(context.Background(), newRequest())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "insert failed")
}

func TestSendEmail_ExistingEmail_SkipsSaveUsesExisting(t *testing.T) {
	writeTemplateFile(t, "templates", "test.html", `Hi!`)
	defer os.RemoveAll("templates")

	existing := &domain.Email{ID: 99, TraceID: "trace-001", StatusType: domain.StatusStart}
	repo := &mockRepo{findResult: existing}
	sender := &mockSender{}
	svc := service.NewEmailService(repo, &mockFetcher{}, sender)

	err := svc.SendEmail(context.Background(), newRequest())
	require.NoError(t, err)
	// Save should NOT be called since email already exists
	assert.Nil(t, repo.saved)
	// UpdateStatus should be called with existing email ID
	require.Len(t, repo.updateCalls, 1)
	assert.Equal(t, int64(99), repo.updateCalls[0].id)
}

func TestSendEmail_AttachmentFetchError_ReturnsError(t *testing.T) {
	writeTemplateFile(t, "templates", "test.html", `Hi!`)
	defer os.RemoveAll("templates")

	repo := &mockRepo{}
	sender := &mockSender{}
	fetcher := &mockFetcher{
		fetchFn: func(_, _ string) (string, func(), error) {
			return "", nil, errors.New("presigned URL expired")
		},
	}
	svc := service.NewEmailService(repo, fetcher, sender)

	req := newRequest()
	req.Attachments = []service.Attachment{{Name: "report.pdf", URL: "https://s3.example.com/report.pdf"}}

	err := svc.SendEmail(context.Background(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetch attachment")
}
