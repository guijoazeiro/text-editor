# Text Editor

Backend in Go for collaborative document editing application (Google Docs style).

## 🚀 Technologies

- **Go 1.25**
- **Gin** - Web framework
- **GORM** - ORM
- **PostgreSQL** - Database
- **Docker** - Containerization


## 🏃 How to Run

### With Docker (Recommended)

1. **Clone the repository**
```bash
git clone https://github.com/guijoazeiro/text-editor.git
cd backend
```

2. **Set up environment variables**
```bash
cp .env.example .env
```

3. **Build and run the application**
```bash
docker compose up --build -d
```
## 📡 Endpoints

### Health Check
```bash
GET /health
```

### Documents

**Create Document**
```bash
POST /api/documents
Content-Type: application/json

{
  "title": "My Document",
  "content": "Initial content..."
}
```

**List Documents**
```bash
GET /api/documents
```

**Get Document**
```bash
GET /api/documents/:id
```

**Update Document**
```bash
PUT /api/documents/:id
Content-Type: application/json

{
  "title": "New title",
  "content": "New content..."
}
```

**Delete Document**
```bash
DELETE /api/documents/:id
```

# 📝 Usage Examples

### Create a document
```bash
curl -X POST http://localhost:8080/api/documents \
  -H "Content-Type: application/json" \
  -d '{"title":"My First Doc","content":"Hello World!"}'
```

### List all documents
```bash
curl http://localhost:8080/api/documents
```

### Get specific document
```bash
curl http://localhost:8080/api/documents/{id}
```

## 🎯 Next Steps

- [ ] JWT authentication system
- [ ] User ↔ Document relationship
- [ ] Permissions system
- [ ] WebSocket for real-time collaboration
- [ ] Document versioning
- [ ] Comments system