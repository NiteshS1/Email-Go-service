package service

import "context"

// EmailService defines the contract for email sending operations.
type EmailService interface {
	SendEmail(ctx context.Context, req SendEmailRequest) error
	SendOTPEmailRequest(req EmailRequest) error
}

// Attachment represents a file attachment for an email.
type Attachment struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// SendEmailRequest is the payload required to send an email.
type SendEmailRequest struct {
	TraceID     string                 `json:"trace_id"`
	TenantID    int64                  `json:"tenant_id"`
	ServiceID   int64                  `json:"service_id"`
	Receiver    string                 `json:"receiver_email"`
	Template    string                 `json:"template"`
	Subject     string                 `json:"subject"`
	Data        map[string]interface{} `json:"data"`
	Attachments []Attachment           `json:"attachments,omitempty"`
}

type EmailRequest struct {
	Body             string `json:"body"`
	Recipient        string `json:"recipient"`
	RecoveryCode     string `json:"recovery_code,omitempty"`
	RecoveryURL      string `json:"recovery_url,omitempty"`
	Subject          string `json:"subject"`
	TemplateType     string `json:"template_type"`
	To               string `json:"to"`
	VerificationCode string `json:"verification_code,omitempty"`
	VerificationURL  string `json:"verification_url,omitempty"`
}
