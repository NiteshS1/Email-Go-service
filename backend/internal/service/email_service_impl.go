package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/smtp"
	"os"

	"github.com/emailservice/internal/domain"
	"github.com/emailservice/internal/fetcher"
	"github.com/emailservice/internal/infrastructure"
	"github.com/emailservice/internal/repository"
	"github.com/emailservice/internal/utils"
)

type emailService struct {
	repo    repository.EmailRepository
	fetcher fetcher.Fetcher
	sender  infrastructure.EmailSender
}

func NewEmailService(
	repo repository.EmailRepository,
	f fetcher.Fetcher,
	sender infrastructure.EmailSender,
) EmailService {
	if f == nil {
		f = fetcher.NewCombinedFetcher()
	}
	return &emailService{repo: repo, fetcher: f, sender: sender}
}

func (s *emailService) SendEmail(ctx context.Context, req SendEmailRequest) error {
	logger := slog.With("trace_id", req.TraceID, "receiver", req.Receiver, "template", req.Template)
	logger.InfoContext(ctx, "processing email request")

	templatePath := "templates/email.html"
	if req.Template != "" {
		templatePath = "templates/" + req.Template + ".html"
	}
	body, err := utils.ParseTemplate(templatePath, req.Data)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	email, findErr := s.repo.FindByTraceID(req.TraceID)
	if findErr != nil {
		return fmt.Errorf("find by trace id: %w", findErr)
	}
	if email == nil {
		email = &domain.Email{
			TraceID:       req.TraceID,
			TenantID:      req.TenantID,
			ServiceID:     req.ServiceID,
			Template:      req.Template,
			Subject:       req.Subject,
			StatusType:    domain.StatusStart,
			ReceiverEmail: req.Receiver,
		}
		if err := s.repo.Save(email); err != nil {
			return fmt.Errorf("save email record: %w", err)
		}
	}

	var infraAttachments []infrastructure.Attachment
	var cleanups []func()
	defer func() {
		for _, c := range cleanups {
			c()
		}
	}()

	for _, att := range req.Attachments {
		if att.Name == "" || att.URL == "" {
			continue
		}
		tempPath, cleanup, fetchErr := s.fetcher.Fetch(att.URL, att.Name)
		if fetchErr != nil {
			logger.WarnContext(ctx, "failed to fetch attachment", "name", att.Name, "error", fetchErr)
			return fmt.Errorf("fetch attachment %q: %w", att.Name, fetchErr)
		}
		cleanups = append(cleanups, cleanup)
		infraAttachments = append(infraAttachments, infrastructure.Attachment{Name: att.Name, Path: tempPath})
	}

	statusType := domain.StatusSuccess
	var errMsg string
	if sendErr := s.sender.Send(req.Receiver, req.Subject, body, infraAttachments); sendErr != nil {
		statusType = domain.StatusFail
		errMsg = sendErr.Error()
		logger.ErrorContext(ctx, "smtp send failed", "error", sendErr)
	} else {
		logger.InfoContext(ctx, "email sent successfully")
	}

	if updateErr := s.repo.UpdateStatus(email.ID, statusType, errMsg); updateErr != nil {
		return fmt.Errorf("update email status: %w", updateErr)
	}
	return nil
}

func (s *emailService) SendOTPEmailRequest(req EmailRequest) error {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	senderEmail := os.Getenv("SMTP_USERNAME")
	appPassword := os.Getenv("SMTP_PASSWORD")

	if smtpHost == "" || smtpPort == "" || senderEmail == "" || appPassword == "" {
		return fmt.Errorf("SMTP configuration is missing")
	}

	auth := smtp.PlainAuth("", senderEmail, appPassword, smtpHost)

	message := fmt.Sprintf(
		"From: %s\nTo: %s\nSubject: %s\nMIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n%s",
		senderEmail, req.To, req.Subject, req.Body,
	)

	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)
	err := smtp.SendMail(addr, auth, senderEmail, []string{req.To}, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	return nil
}
