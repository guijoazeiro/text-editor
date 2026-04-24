-- Migration 000011: Replace wall-clock timestamp with Lamport logical clock on yjs_updates.
--
-- Changes:
--   • DROP  clock      BIGINT  (wall-clock, incorrect for causal ordering)
--   • ADD   lamport_ts BIGINT  (logical Lamport clock, extracted from the Yjs update binary)
--   • ADD   client_id  BIGINT  (Yjs client ID that produced the update, for audit / debugging)
--
-- Note: existing rows (if any) receive lamport_ts = 0, client_id = 0.
-- The table is typically empty at this point because it was truncated to fix
-- the prior duplication bug, so the DEFAULT values are never observed.

ALTER TABLE yjs_updates
    DROP COLUMN IF EXISTS clock,
    ADD COLUMN  lamport_ts BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN  client_id  BIGINT NOT NULL DEFAULT 0;

-- Index for ordered retrieval per document (replaces idx_yjs_updates_clock).
CREATE INDEX IF NOT EXISTS idx_yjs_updates_document_lamport
    ON yjs_updates (document_id, lamport_ts);
