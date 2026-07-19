package domain

type EmailRequest struct {
	Body         string `gorm:"column:body;not null"`
	Recipient    string `gorm:"column:recipient;not null"`
	RecoveryCode string `gorm:"column:recovery_code;not null"`
	RecoveryURL  string `gorm:"column:recovery_url;not null"`
	Subject      string `gorm:"column:subject;not null"`
	TemplateType string `gorm:"column:template_type;not null"`
	To           string `gorm:"column:to;not null"`
	VerificationCode string `gorm:"column:verification_code;not null"`
	VerificationURL  string `gorm:"column:verification_url;not null"`
}
