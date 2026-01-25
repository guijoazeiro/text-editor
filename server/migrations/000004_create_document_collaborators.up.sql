CREATE TYPE permission_type AS ENUM ('owner', 'editor', 'viewer');

CREATE TABLE IF NOT EXISTS document_collaborators (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission permission_type NOT NULL DEFAULT 'viewer',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(document_id, user_id)
);

CREATE INDEX idx_document_collaborators_document_id ON document_collaborators(document_id);
CREATE INDEX idx_document_collaborators_user_id ON document_collaborators(user_id);