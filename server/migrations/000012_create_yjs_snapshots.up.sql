CREATE TABLE IF NOT EXISTS yjs_snapshots (
    id          UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID    NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    snapshot    BYTEA   NOT NULL,
    -- LamportTS of the highest update included in this snapshot.
    -- Updates in yjs_updates with lamport_ts > this value are the "delta"
    -- that must be applied on top of the snapshot to reconstruct current state.
    lamport_ts  BIGINT  NOT NULL DEFAULT 0,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Only one active snapshot per document (UPSERT pattern).
    CONSTRAINT uq_yjs_snapshots_document UNIQUE (document_id)
);

CREATE INDEX IF NOT EXISTS idx_yjs_snapshots_document_id ON yjs_snapshots (document_id);
