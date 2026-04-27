DROP INDEX IF EXISTS idx_yjs_updates_document_lamport;

ALTER TABLE yjs_updates
    DROP COLUMN IF EXISTS lamport_ts,
    DROP COLUMN IF EXISTS client_id,
    ADD COLUMN  clock BIGINT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_yjs_updates_clock
    ON yjs_updates (clock);
