package infrastructure

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/smtp"
	"os"
	"path/filepath"

	"github.com/emailservice/internal/config"
)

// Attachment represents a file to attach to an email.
type Attachment struct {
	Name string
	Path string
}

// EmailSender is the interface that wraps email sending functionality.
// Using an interface allows easy mocking in unit tests.
type EmailSender interface {
	Send(to, subject, body string, attachments []Attachment) error
}

// SMTPSender sends emails via SMTP using config provided at construction.
type SMTPSender struct {
	from     string
	password string
	host     string
	port     int
}

// NewSMTPSender creates a new SMTPSender from the provided config.
func NewSMTPSender(cfg *config.Config) *SMTPSender {
	return &SMTPSender{
		from:     cfg.SMTPFrom,
		password: cfg.SMTPPassword,
		host:     cfg.SMTPHost,
		port:     cfg.SMTPPort,
	}
}

// Send composes and transmits an email with optional attachments.
func (s *SMTPSender) Send(to, subject, body string, attachments []Attachment) error {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	boundary := writer.Boundary()

	buf.WriteString("From: " + s.from + "\r\n")
	buf.WriteString("To: " + to + "\r\n")
	buf.WriteString("Subject: " + subject + "\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: multipart/mixed; boundary=" + boundary + "\r\n\r\n")

	part, err := writer.CreatePart(map[string][]string{"Content-Type": {"text/html; charset=UTF-8"}})
	if err != nil {
		return fmt.Errorf("create html part: %w", err)
	}
	if _, err = part.Write([]byte(body)); err != nil {
		return fmt.Errorf("write html body: %w", err)
	}

	for _, att := range attachments {
		data, err := os.ReadFile(att.Path)
		if err != nil {
			return fmt.Errorf("read attachment %s: %w", att.Name, err)
		}
		ext := filepath.Ext(att.Name)
		contentType := contentTypeByExt(ext)
		attPart, err := writer.CreatePart(map[string][]string{
			"Content-Type":              {contentType + "; name=\"" + att.Name + "\""},
			"Content-Disposition":       {"attachment; filename=\"" + att.Name + "\""},
			"Content-Transfer-Encoding": {"base64"},
		})
		if err != nil {
			return fmt.Errorf("create part for %s: %w", att.Name, err)
		}
		enc := base64.NewEncoder(base64.StdEncoding, attPart)
		if _, err = enc.Write(data); err != nil {
			return fmt.Errorf("encode attachment %s: %w", att.Name, err)
		}
		if err = enc.Close(); err != nil {
			return fmt.Errorf("close encoder for %s: %w", att.Name, err)
		}
	}

	if err = writer.Close(); err != nil {
		return fmt.Errorf("close multipart writer: %w", err)
	}

	auth := smtp.PlainAuth("", s.from, s.password, s.host)
	return smtp.SendMail(
		fmt.Sprintf("%s:%d", s.host, s.port),
		auth,
		s.from,
		[]string{to},
		buf.Bytes(),
	)
}

func contentTypeByExt(ext string) string {
	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	default:
		return "application/octet-stream"
	}
}
