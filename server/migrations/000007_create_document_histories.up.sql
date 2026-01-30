CREATE TYPE action_type AS ENUM ('created', 'updated', 'title_changed', 'content_changed');

CREATE TABLE IF NOT EXISTS document_histories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    action action_type NOT NULL,
    changes JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_document_histories_document_id ON document_histories(document_id);
CREATE INDEX idx_document_histories_created_at ON document_histories(created_at DESC);