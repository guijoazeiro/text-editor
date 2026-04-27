# Docs Editor — Real-Time Collaborative Editor

A collaborative document editor inspired by Google Docs, built from scratch to learn and demonstrate real distributed synchronization technologies (**CRDT via Yjs**) and real-time communication (**WebSockets**).

> 📖 [Leia em Português](./README.pt.md)

---

## Stack

| Layer | Technology |
|-------|-----------|
| Frontend | Next.js 14, TypeScript, Tailwind CSS |
| Editor | TipTap v2 + collaboration extensions |
| CRDT | Yjs + y-protocols |
| Real-time | WebSocket (gorilla/websocket) |
| Backend | Go 1.22 + Gin |
| Database | PostgreSQL + golang-migrate |
| Infra | Docker + docker-compose |

---

## Features

### Authentication
- Sign-up and login with JWT (24h expiry)
- bcrypt password hashing
- Route protection middleware (HTTP and WebSocket)
- WebSocket auth via **subprotocol** (`Sec-WebSocket-Protocol`) — safer than query params; token never appears in server logs

### Documents
- Full CRUD with automatic owner assignment
- `content_format` field to distinguish legacy plain text (`"text"`) from rich text (`"tiptap"`)
- `User` relation preloaded on all responses

### Permission System
- Three levels: `owner`, `editor`, `viewer`
- Owner: all operations
- Editor: can edit, cannot delete or manage collaborators
- Viewer: read-only

### Collaboration
- Add collaborators by email
- Update collaborator permissions
- Remove collaborators
- Public share links with configurable permissions and optional expiry

### Notifications
- When added as a collaborator
- When permission changes
- When a collaborator edits a document (owner receives)
- Mark as read individually or in bulk
- Unread counter

### History & Versioning
- Edit history: who edited, when, and what changed (diff)
- History filters: by user, action type, date
- Versioning with full document snapshots
- Restore a previous version (saves the current version first)
- Compare two versions (diff)

### WebSocket — Real-time Presence
- Rooms per document: each document has its own isolated room
- Presence tracking: automatic join/leave on connect/disconnect
- Event broadcast to all clients in the room
- Automatic reconnection with exponential backoff (up to 5 attempts)
- JWT auth via WebSocket subprotocol

### CRDT with Yjs
**Why CRDT?** Operational Transformation (OT, used in the original Google Docs) resolves simultaneous edit conflicts via mathematical transformations that require a central server as an arbiter. CRDT solves the same problem by design — each operation is commutative and idempotent, so two clients applying the same operations in any order reach the same result, without a central arbiter.

**Architecture:**
```
Client A (Yjs doc) ──┐
                      ├── WebSocket ──► Go Backend (relay) ──► PostgreSQL
Client B (Yjs doc) ──┘                     │
                                           ├── yjs_updates table
                                           └── yjs_snapshots table
```

The Go backend **does not need to understand** Yjs content — it is a pure relay. This is intentional: merge logic lives in the clients.

**Sync protocol:**
1. Client connects → sends `SyncStep1` (state vector: which operations it already has)
2. Peer receives `SyncStep1` → responds with `SyncStep2` (diff: operations the other doesn't have)
3. Client receives `SyncStep2` → applies to local doc, marked as `synced`
4. Any local edit → sends `Update` to all peers via relay
5. Reconnection → repeats from step 1

**Persistence:** The backend saves each Yjs update in the `yjs_updates` table (bytea). When a new client connects, the hub sends all saved updates so it can reconstruct the full state — the document persists even with no peer online.

**Compaction:** A Node.js worker (`compactor/`) runs `Y.mergeUpdates` to merge individual updates into a single snapshot when a count or size threshold is reached. Compaction also runs when the last peer disconnects. This keeps `yjs_updates` from growing unboundedly.

**Rich text with TipTap:**
- TipTap uses `Y.XmlFragment("content")` as the underlying CRDT (not `Y.Text`)
- `Y.XmlFragment` is a collaborative XML tree — each node (paragraph, heading, bold) is an independent CRDT operation
- Two users can format the same text simultaneously without conflict
- `@tiptap/extension-collaboration` connects ProseMirror (TipTap's engine) to Yjs automatically

### Visual Awareness
- Real-time remote cursors: colored caret line at each user's cursor position
- Name badges appear for 2.5s on cursor move
- Colored selections from other users (semi-transparent highlight)
- Tooltips with name and "editing" status on avatars
- Pulsing dot on avatar when the user has an active cursor

---

## Project Structure

```
text-editor/
├── client/                     # Next.js frontend
│   └── src/
│       ├── app/
│       │   ├── dashboard/      # Document list
│       │   ├── editor/[id]/    # Main editor
│       │   ├── login/
│       │   └── signup/
│       ├── components/
│       │   ├── EditorToolbar.tsx     # TipTap formatting toolbar
│       │   ├── Navbar.tsx
│       │   ├── RemoteCursors.tsx     # Remote cursors and selections
│       │   ├── TiptapEditor.tsx      # Editor wrapper
│       │   └── UserPresence.tsx      # Online avatars
│       ├── hooks/
│       │   ├── useWebSocket.ts       # WebSocket connection + presence
│       │   └── useYjsEditor.ts       # Yjs doc + provider + sync state
│       ├── lib/
│       │   ├── api.ts                # HTTP client (axios)
│       │   ├── websocket.ts          # WebSocketClient class
│       │   └── yjs-provider.ts       # Yjs sync protocol
│       └── store/
│           └── authStore.ts          # Zustand (JWT + user)
│
└── server/                     # Go backend
    ├── cmd/server/main.go
    ├── compactor/              # Node.js worker for Yjs merge operations
    │   └── index.js            # /compact and /state-vector endpoints
    ├── internal/
    │   ├── auth/               # JWT + bcrypt
    │   ├── config/
    │   ├── database/           # Connection + migrations runner
    │   ├── handlers/           # HTTP handlers (Gin)
    │   ├── middleware/         # Auth middleware
    │   ├── models/             # GORM models
    │   ├── services/           # Business logic
    │   └── websocket/          # Hub + Client (gorilla/websocket)
    ├── migrations/             # Versioned SQL (golang-migrate)
    │   ├── 000001 → documents
    │   ├── 000002 → users
    │   ├── 000003 → user_id on documents
    │   ├── 000004 → document_collaborators
    │   ├── 000005 → document_share_links
    │   ├── 000006 → notifications
    │   ├── 000007 → document_histories
    │   ├── 000008 → document_versions
    │   ├── 000009 → yjs_updates
    │   ├── 000010 → content_format on documents
    │   ├── 000011 → lamport_ts + client_id on yjs_updates
    │   ├── 000012 → yjs_snapshots
    │   └── 000013 → yjs_snapshot on document_versions
    └── pkg/response/           # Standardized responses
```

---

## Running Locally

### Prerequisites
- Docker and docker-compose
- Node.js 18+

### Backend

```bash
cd server

# Copy environment variables
cp .env.example .env

# Start PostgreSQL + backend (migrations run automatically)
docker-compose up -d

# View logs
docker-compose logs -f backend
```

### Frontend

```bash
cd client

# Install dependencies
npm install

# Create .env.local
echo "NEXT_PUBLIC_API_URL=http://localhost:8080" > .env.local
echo "NEXT_PUBLIC_WS_URL=ws://localhost:8080" >> .env.local

# Start development server
npm run dev
```

Open `http://localhost:3000`.

---

## API — Main Endpoints

### Auth
| Method | Route | Description |
|--------|-------|-------------|
| POST | `/api/auth/signup` | Create account |
| POST | `/api/auth/login` | Login (returns JWT) |
| GET | `/api/auth/me` | Authenticated user data |

### Documents
| Method | Route | Description |
|--------|-------|-------------|
| POST | `/api/documents` | Create document |
| GET | `/api/documents` | List (owned + shared) |
| GET | `/api/documents/:id` | Fetch by ID + permission |
| PUT | `/api/documents/:id` | Update |
| DELETE | `/api/documents/:id` | Delete (owner only) |

### Collaboration
| Method | Route | Description |
|--------|-------|-------------|
| POST | `/api/documents/:id/collaborators` | Add by email |
| GET | `/api/documents/:id/collaborators` | List |
| PUT | `/api/documents/:id/collaborators/:user_id` | Update permission |
| DELETE | `/api/documents/:id/collaborators/:user_id` | Remove |
| POST | `/api/documents/:id/share-link` | Create public link |
| DELETE | `/api/documents/:id/share-link` | Remove link |

### History & Versions
| Method | Route | Description |
|--------|-------|-------------|
| GET | `/api/documents/:id/history` | History with filters |
| GET | `/api/documents/:id/versions` | List versions |
| GET | `/api/documents/:id/versions/:n` | Fetch specific version |
| POST | `/api/documents/:id/versions/:n/restore` | Restore version |
| GET | `/api/documents/:id/versions/compare?v1=N&v2=M` | Compare versions |

### Notifications
| Method | Route | Description |
|--------|-------|-------------|
| GET | `/api/notifications` | List + unread count |
| PUT | `/api/notifications/:id/read` | Mark as read |
| PUT | `/api/notifications/read-all` | Mark all as read |
| DELETE | `/api/notifications/:id` | Delete |

### WebSocket
| Route | Description |
|-------|-------------|
| `WS /ws/documents/:id` | Connect to document (auth via subprotocol) |

### CRDT (Yjs)
| Method | Route | Description |
|--------|-------|-------------|
| GET | `/api/documents/:id/yjs-updates` | Raw CRDT updates |
| GET | `/api/documents/:id/yjs-state-vector` | Current state vector (reconnection optimization) |

---

## Technical Decisions

**Why Go on the backend?** Native performance for WebSockets with many simultaneous connections. Goroutines are far lighter than threads — each WebSocket connection runs in its own goroutine with minimal overhead.

**Why Yjs and not a custom CRDT?** Yjs is battle-tested and used in production by Notion, Figma, and others. Implementing CRDT from scratch (LSEQ, RGA, Logoot) is a dissertation-level project. The learning here is in *understanding and integrating* Yjs, not reinventing proven algorithms.

**Why doesn't the Go backend need to understand Yjs?** CRDT is peer-to-peer by nature. The server is just a relay — it forwards bytes from one client to the others. This keeps the backend stateless with respect to document content and easy to scale horizontally (with a message broker like Redis Pub/Sub between instances).

**Versioned migrations vs AutoMigrate:** GORM's AutoMigrate is convenient but has no version control, no rollback, and is not recommended for production. `golang-migrate` with numbered SQL files gives full traceability — each schema change is an `up.sql` + `down.sql` file committed to the repository.

**WebSocket auth via subprotocol:** Browsers do not allow custom headers on WebSocket connections. The common (and insecure) solution is passing the token as a query param — it appears in server logs, proxy logs, and browser history. The subprotocol (`Sec-WebSocket-Protocol`) is a standard WebSocket header sent during the handshake without appearing in the URL.

**CRDT compaction with a Node.js worker:** Go has no native Yjs implementation. To merge updates (`Y.mergeUpdates`) and generate state vectors, a Node.js worker (`compactor/`) exposes HTTP endpoints consumed by the Go backend. This avoids WebAssembly and keeps each responsibility in its natural language. The compactor applies a dual threshold policy: compact when updates exceed a count limit **or** a total size limit. Idle compaction also runs when the last peer disconnects.

**Version restore via `document-content-reset`:** Restoring a version using a binary CRDT snapshot does not work for time-travel — `Y.applyUpdate` is additive, and operations with a higher Lamport clock always win. The correct approach is to broadcast the version's JSON content via WebSocket (`document-content-reset`) and have each client call `editor.commands.setContent()`. This generates new Yjs operations with the current maximum clock, properly overwriting the old content in every peer's Y.Doc. The server-side CRDT state is also cleared on restore so that newly connecting clients start from a clean slate and seed from the restored `documents.content`.

**Causal ordering with Lamport timestamps:** The backend decodes the binary Yjs v1 update format in Go (without instantiating a Y.Doc) to extract the `LamportTS` and `ClientID` of each update. These are indexed in `yjs_updates` to guarantee causal ordering when replaying history to new peers, and to correctly determine the delta to send during reconnection.