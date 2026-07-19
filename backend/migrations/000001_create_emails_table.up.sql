-- Create emails table for email service
CREATE TABLE IF NOT EXISTS emails (
    id              BIGSERIAL PRIMARY KEY,
    trace_id        VARCHAR(255) NOT NULL,
    tenant_id       BIGINT NOT NULL,
    service_id      BIGINT NOT NULL,
    template        VARCHAR(255) NOT NULL,
    subject         TEXT NOT NULL,
    status_type     VARCHAR(50) NOT NULL,
    receiver_email  VARCHAR(255) NOT NULL,
    error_message   TEXT,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index for lookups by trace_id
CREATE INDEX IF NOT EXISTS idx_emails_trace_id ON emails (trace_id);

-- Optional: index for tenant/service filtering
CREATE INDEX IF NOT EXISTS idx_emails_tenant_service ON emails (tenant_id, service_id);
