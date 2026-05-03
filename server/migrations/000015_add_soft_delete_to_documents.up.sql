-- Add soft delete support to documents table.
-- GORM's soft delete uses deleted_at TIMESTAMPTZ NULL; records with a non-NULL
-- value are excluded from normal queries via a WHERE deleted_at IS NULL clause.
ALTER TABLE documents ADD COLUMN deleted_at TIMESTAMPTZ NULL;

-- Partial index: only non-deleted documents are indexed, keeping scans fast.
CREATE INDEX idx_documents_deleted_at ON documents(deleted_at) WHERE deleted_at IS NULL;
