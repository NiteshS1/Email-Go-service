package domain

import "time"

type StatusType string

const (
	StatusStart   StatusType = "start"
	StatusSuccess StatusType = "success"
	StatusFail    StatusType = "fail"
)

type Email struct {
	ID            int64      `gorm:"column:id;primaryKey;autoIncrement"`
	TraceID       string     `gorm:"column:trace_id;not null;index"`
	TenantID      int64      `gorm:"column:tenant_id;not null"`
	ServiceID     int64      `gorm:"column:service_id;not null"`
	Template      string     `gorm:"column:template;not null"`
	Subject       string     `gorm:"column:subject;not null"`
	StatusType    StatusType `gorm:"column:status_type;not null"`
	ReceiverEmail string     `gorm:"column:receiver_email;not null"`
	ErrorMessage  string     `gorm:"column:error_message"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime"`
}

// TableName overrides the table name for GORM
func (Email) TableName() string {
	return "emails"
}
