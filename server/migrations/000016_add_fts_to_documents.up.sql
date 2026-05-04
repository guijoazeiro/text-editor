-- Add a tsvector column for efficient full-text search across title and content.
-- We use a PostgreSQL generated column (STORED) so Postgres maintains it
-- automatically on every INSERT/UPDATE — no application-level trigger needed.
ALTER TABLE documents
  ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
      setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
      setweight(to_tsvector('english', coalesce(content, '')), 'B')
    ) STORED;

-- GIN index: required for fast @@ operator lookups on the tsvector column.
CREATE INDEX idx_documents_search_vector ON documents USING GIN(search_vector);
