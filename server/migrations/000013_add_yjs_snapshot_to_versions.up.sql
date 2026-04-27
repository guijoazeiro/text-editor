ALTER TABLE document_versions
    ADD COLUMN IF NOT EXISTS yjs_snapshot BYTEA;
