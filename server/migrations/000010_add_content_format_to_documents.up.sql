ALTER TABLE documents ADD COLUMN IF NOT EXISTS content_format VARCHAR(20) NOT NULL DEFAULT 'text';
CREATE INDEX IF NOT EXISTS idx_documents_content_format ON documents(content_format);
