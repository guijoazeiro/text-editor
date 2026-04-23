CREATE TABLE IF NOT EXISTS yjs_updates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    update BYTEA NOT NULL,
    clock BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_yjs_updates_document_id ON yjs_updates(document_id);
CREATE INDEX idx_yjs_updates_clock ON yjs_updates(clock);
CREATE INDEX idx_yjs_updates_document_clock ON yjs_updates(document_id, clock);
