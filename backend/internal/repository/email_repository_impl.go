package repository

import (
	"github.com/emailservice/internal/domain"
	"gorm.io/gorm"
)

type emailRepository struct {
	db *gorm.DB
}

func NewEmailRepository(db *gorm.DB) EmailRepository {
	return &emailRepository{db: db}
}

func (r *emailRepository) Save(email *domain.Email) error {
	return r.db.Create(email).Error
}

func (r *emailRepository) FindByTraceID(traceID string) (*domain.Email, error) {
	var e domain.Email
	err := r.db.Where("trace_id = ?", traceID).First(&e).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &e, nil
}

func (r *emailRepository) UpdateStatus(id int64, statusType domain.StatusType, errorMessage string) error {
	return r.db.Model(&domain.Email{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status_type":   statusType,
			"error_message": errorMessage,
		}).Error
}
