-- Drop emails table and indexes
DROP INDEX IF EXISTS idx_emails_tenant_service;
DROP INDEX IF EXISTS idx_emails_trace_id;
DROP TABLE IF EXISTS emails;
