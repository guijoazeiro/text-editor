DROP INDEX IF EXISTS idx_documents_content_format;
ALTER TABLE documents DROP COLUMN IF EXISTS content_format;
