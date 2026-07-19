package repository

import "github.com/emailservice/internal/domain"

type EmailRepository interface {
	Save(email *domain.Email) error
	FindByTraceID(traceID string) (*domain.Email, error)
	UpdateStatus(id int64, statusType domain.StatusType, errorMessage string) error
}
