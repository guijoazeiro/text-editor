ALTER TABLE documents 
ADD COLUMN user_id UUID REFERENCES users(id) ON DELETE CASCADE;

CREATE INDEX idx_documents_user_id ON documents(user_id);