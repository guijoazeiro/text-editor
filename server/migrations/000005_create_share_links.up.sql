CREATE TABLE IF NOT EXISTS document_share_links (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    token VARCHAR(255) UNIQUE NOT NULL,
    permission permission_type NOT NULL DEFAULT 'viewer',
    expires_at TIMESTAMP WITH TIME ZONE,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(document_id)
);

CREATE INDEX idx_share_links_token ON document_share_links(token);
CREATE INDEX idx_share_links_document_id ON document_share_links(document_id);