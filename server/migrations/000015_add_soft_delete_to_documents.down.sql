DROP INDEX IF EXISTS idx_documents_deleted_at;
ALTER TABLE documents DROP COLUMN IF EXISTS deleted_at;
