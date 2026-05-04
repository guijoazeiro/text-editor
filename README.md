# Docs Editor — Real-Time Collaborative Editor

A collaborative document editor inspired by Google Docs, built from scratch to learn and demonstrate real distributed synchronization technologies (**CRDT via Yjs**) and real-time communication (**WebSockets**).

> 📖 [Leia em Português](./README.pt.md)

---

## Stack

| Layer | Technology |
|-------|-----------|
| Frontend | Next.js 16, TypeScript, Tailwind CSS v4 |
| Editor | TipTap v2 + collaboration extensions |
| CRDT | Yjs + y-protocols |
| Real-time | WebSocket (gorilla/websocket) |
| Backend | Go 1.22 + Gin |
| Database | PostgreSQL + golang-migrate |
| Infra | Docker + docker-compose |

---

## Features

### Authentication
- Sign-up and login with JWT (access token 24h, refresh token 30 days via HttpOnly cookie)
- bcrypt password hashing
- Route protection middleware (HTTP and WebSocket)
- WebSocket auth via **subprotocol** (`Sec-WebSocket-Protocol`) — safer than query params; token never appears in server logs
- JWT expiry check on store rehydration — expired sessions are cleared automatically on page load
- **Token refresh:** `axios` interceptor retries failed 401 requests after calling `POST /api/auth/refresh`; WebSocket client handles close code `4001` (token expired on long-lived connection) and refreshes silently without interrupting the session
- **Rate limiting on auth endpoints:** login limited to 5 req/min per IP, signup to 3 req/min per IP (token bucket, in-memory, with automatic cleanup)
- `PATCH /api/auth/me` — update display name

### Documents
- Full CRUD with automatic owner assignment
- `content_format` field to distinguish legacy plain text (`"text"`) from rich text (`"tiptap"`)
- `User` relation preloaded on all responses
- **Soft delete:** `DELETE /api/documents/:id` sets `deleted_at` instead of removing the row; documents are recoverable via the Trash panel for 30 days before being permanently purged
- **Trash & Restore:** `GET /api/documents/trash` lists soft-deleted documents; `POST /api/documents/:id/restore` recovers them
- **Full-text search:** `GET /api/documents?q=term` uses PostgreSQL `tsvector` + `websearch_to_tsquery` with a GIN index; results are ranked by `ts_rank` (title weighted higher than content)
- **Pagination:** `?page=N&limit=N` on the document list

### Permission System
- Three levels: `owner`, `editor`, `viewer`
- Owner: all operations
- Editor: can edit, cannot delete or manage collaborators
- Viewer: read-only

### Collaboration
- Add collaborators by email with permission selection (editor / viewer)
- Update collaborator permissions inline
- Remove collaborators
- Public share links with configurable permissions and optional expiry

### Notifications
- When added as a collaborator
- When permission changes
- When a collaborator edits a document (owner receives)
- Mark as read individually or in bulk
- Unread counter badge in the navbar
- Real-time dropdown accessible from any page
- **Real-time push via WebSocket:** notifications are delivered instantly over the existing WS connection (`notification:new` message) without polling — the `Hub` maintains a `UserClients` index to target all active connections for a specific user

### History & Versioning
- Edit history: who edited, when, and what changed (diff)
- History filters: by user, action type, date
- Versioning with full document snapshots
- Restore a previous version (saves the current version first)
- Compare two versions (diff)
- Version history panel in the editor (last 20 versions, slide-in from right)

### WebSocket — Real-time Presence
- Rooms per document: each document has its own isolated room
- Presence tracking: automatic join/leave on connect/disconnect
- Event broadcast to all clients in the room
- Automatic reconnection with exponential backoff (up to 5 attempts)
- JWT auth via WebSocket subprotocol
- **JWT expiry check on long-lived connections:** a 15-minute ticker in `WritePump` parses the raw JWT (`ParseUnverified`) and sends close code `4001` if expired; the client intercepts `4001`, calls `POST /api/auth/refresh`, and reconnects automatically

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

### UI & Design System
- **Dark / Light mode** with a persistent toggle (localStorage via Zustand)
- CSS variable–based theming — all components use `var(--bg-base)`, `var(--text-primary)`, etc.
- **Inter** font (Google Fonts via `next/font`)
- Skeleton loading states on the document dashboard
- Two-column login and signup pages (dark branding panel + form panel)
- Notification bell with unread badge in the navbar
- User profile page (`/profile`) with inline name editing and sign-out
- Toast notifications for all async actions (success / error)
- Version history slide-in panel with restore confirmation modal
- Collaborators modal with invite by email, permission selector, public link management
- **Dashboard search bar** with 350 ms debounce — filters documents by title or content via FTS
- **Trash panel** in the dashboard — shows soft-deleted documents with a Restore button; permanent purge after 30 days
- **Soft-delete button** on each document card (appears on hover, single click, no confirmation needed since it's reversible)

---

## Project Structure

```
text-editor/
├── client/                     # Next.js frontend
│   └── src/
│       ├── app/
│       │   ├── dashboard/      # Document list with skeleton loading
│       │   ├── editor/[id]/    # Main editor
│       │   ├── login/          # Two-column layout
│       │   ├── signup/         # Two-column layout
│       │   └── profile/        # User profile page
│       ├── components/
│       │   ├── editor/
│       │   │   ├── EditorToolbar.tsx       # TipTap formatting toolbar
│       │   │   ├── VersionHistory.tsx      # Slide-in version history panel
│       │   │   └── CollaboratorsModal.tsx  # Share & collaborators modal
│       │   ├── layout/
│       │   │   ├── Navbar.tsx              # With theme toggle, notifications, avatar
│       │   │   └── ThemeProvider.tsx       # Applies .dark class to <html>
│       │   ├── notifications/
│       │   │   └── NotificationBell.tsx    # Dropdown with unread badge
│       │   ├── presence/
│       │   │   ├── RemoteCursors.tsx       # Remote cursors and selections
│       │   │   └── UserPresence.tsx        # Online avatars
│       │   └── ui/
│       │       ├── Button.tsx
│       │       ├── Input.tsx               # Theme-aware, with password toggle
│       │       ├── Modal.tsx
│       │       ├── Skeleton.tsx            # DocumentCardSkeleton
│       │       └── Toast.tsx
│       ├── hooks/
│       │   ├── useWebSocket.ts             # WebSocket connection + presence
│       │   ├── useVersionHistory.ts        # Fetch + restore versions
│       │   └── useYjsEditor.ts             # Yjs doc + provider + sync state
│       ├── lib/
│       │   ├── api.ts                      # HTTP client (axios) — all API calls
│       │   ├── axios.ts                    # Axios instance + auth interceptor
│       │   ├── utils.ts                    # cn() helper
│       │   ├── websocket.ts                # WebSocketClient class
│       │   └── yjs-provider.ts             # Yjs sync protocol
│       ├── store/
│       │   ├── authStore.ts                # Zustand (JWT + user + isHydrated)
│       │   ├── themeStore.ts               # Dark/light mode (persisted)
│       │   └── toastStore.ts               # Global toasts
│       └── types/
│           ├── collaborator.ts
│           ├── document.ts
│           ├── notification.ts
│           ├── user.ts
│           ├── version.ts
│           └── index.ts
│
└── server/                     # Go backend
    ├── cmd/server/main.go
    ├── compactor/              # Node.js worker for Yjs merge operations
    │   └── index.js            # /compact, /state-vector, /health endpoints
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
    │   ├── 000013 → yjs_snapshot on document_versions
    │   ├── 000014 → refresh_token on users
    │   ├── 000015 → soft delete (deleted_at) on documents
    │   └── 000016 → tsvector FTS column + GIN index on documents
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
| POST | `/api/auth/login` | Login (returns access JWT + sets refresh cookie) |
| POST | `/api/auth/refresh` | Refresh access token via HttpOnly cookie |
| GET | `/api/auth/me` | Authenticated user data |
| PATCH | `/api/auth/me` | Update display name |
| POST | `/api/auth/logout` | Revoke refresh token |

### Documents
| Method | Route | Description |
|--------|-------|-------------|
| POST | `/api/documents` | Create document |
| GET | `/api/documents?q=&page=&limit=` | List (owned + shared), with FTS search |
| GET | `/api/documents/trash` | List soft-deleted documents |
| GET | `/api/documents/:id` | Fetch by ID + permission |
| PUT | `/api/documents/:id` | Update |
| DELETE | `/api/documents/:id` | Soft delete — moves to Trash |
| POST | `/api/documents/:id/restore` | Restore from Trash |

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

**CSS variable–based theming:** Instead of Tailwind dark-mode utility classes scattered across components, all color tokens are defined as CSS variables in `globals.css` (e.g. `--bg-base`, `--text-primary`, `--border`). The `ThemeProvider` component toggles the `.dark` class on `<html>`, which swaps the variable values. This means every component is theme-aware by default, and adding a third theme (e.g. high-contrast) requires changing only the CSS variable definitions.

**Type-safe frontend architecture:** All API response shapes are defined in `src/types/` (document, user, collaborator, notification, version). The `authStore` (Zustand + persist) validates the JWT expiry on rehydration and clears expired sessions before any route guard runs — preventing stale auth state after a long idle period.

**Refresh token with SHA-256:** Refresh tokens are random high-entropy strings — bcrypt is intentionally slow and designed for low-entropy passwords. Tokens are stored as SHA-256 hashes (`hex.EncodeToString(sha256.Sum256(token))`) which is fast, collision-resistant, and appropriate for pre-validated secrets.

**Real-time notifications via WebSocket:** The `Hub` maintains a `UserClients map[uuid.UUID]map[*Client]bool` index alongside the per-document index. `NotificationService` holds a `NotificationHub` interface (to avoid circular imports) and calls `SendToUser` after each DB insert, delivering the notification instantly to all active connections of the target user. The frontend `NotificationBell` listens for a global `notification:new` `CustomEvent` dispatched by `yjs-provider.ts`.

**JWT expiry on long-lived WebSocket connections:** A short-lived HTTP JWT is fine for API calls (each request re-validates), but a WebSocket connection can stay open for hours. A 15-minute `tokenTicker` in `WritePump` calls `jwt.ParseUnverified` (no key needed — just reads the `exp` claim) and sends close code `4001` if expired. The frontend `WebSocketClient` catches `4001`, calls `POST /api/auth/refresh`, updates the token in `authStore`, dispatches `ws:token-refreshed`, and reconnects — no user action required.

**Soft delete and PostgreSQL FTS:** Documents are never immediately deleted — `DELETE /api/documents/:id` sets `deleted_at` (GORM `DeletedAt` field, automatically filtered from all queries). A background goroutine purges records older than 30 days. Full-text search uses a `tsvector` generated column (`GENERATED ALWAYS AS ... STORED`) combining title (weight A) and content (weight B), with a GIN index for fast `@@` lookups and `ts_rank` ordering. `websearch_to_tsquery` is used instead of `plainto_tsquery` to support quoted phrases and exclusions.

**Docker `.dockerignore`:** The `compactor/` directory contains a Node.js worker with its own `node_modules`. Without `.dockerignore`, `docker build` transfers `node_modules` to the daemon — causing failures due to file names with special characters (e.g. `@scope/pkg`, `0ecdsa-generate-keypair`). The `.dockerignore` excludes `compactor/node_modules`, `bin/`, and `.env` files.