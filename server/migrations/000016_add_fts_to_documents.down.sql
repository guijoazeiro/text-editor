DROP INDEX IF EXISTS idx_documents_search_vector;
ALTER TABLE documents DROP COLUMN IF EXISTS search_vector;
