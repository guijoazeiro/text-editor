# Docs Editor — Editor Colaborativo em Tempo Real

Editor de documentos colaborativo inspirado no Google Docs, construído do zero com foco em aprender e demonstrar tecnologias reais de sincronização distribuída (**CRDT via Yjs**) e comunicação em tempo real (**WebSockets**).

---

## Stack

| Camada | Tecnologia |
|--------|-----------|
| Frontend | Next.js 14, TypeScript, Tailwind CSS |
| Editor | TipTap v2 + extensões de colaboração |
| CRDT | Yjs + y-protocols |
| Tempo real | WebSocket (gorilla/websocket) |
| Backend | Go 1.22 + Gin |
| Banco | PostgreSQL + golang-migrate |
| Infra | Docker + docker-compose |

---

## Funcionalidades implementadas

### Autenticação
- Registro e login com JWT (24h de validade)
- Senhas com bcrypt
- Middleware de proteção de rotas (HTTP e WebSocket)
- Autenticação WebSocket via **subprotocol** (`Sec-WebSocket-Protocol`) — mais seguro que query params, token não aparece em logs

### Documentos
- CRUD completo com owner automático
- Campo `content_format` para distinguir texto legado (`"text"`) de rich text (`"tiptap"`)
- Preload de relação `User` em todas as respostas

### Sistema de permissões
- Três níveis: `owner`, `editor`, `viewer`
- Owner: todas as operações
- Editor: pode editar, não pode deletar nem gerenciar colaboradores
- Viewer: somente leitura

### Colaboração
- Adicionar colaboradores por email
- Atualizar permissão de colaboradores
- Remover colaboradores
- Links públicos de compartilhamento com permissão configurável e expiração opcional

### Notificações
- Ao ser adicionado como colaborador
- Ao ter permissão alterada
- Ao colaborador editar o documento (owner recebe)
- Marcar como lida individualmente ou em lote
- Contador de não lidas

### Histórico e versionamento
- Histórico de edições: quem editou, quando e o que mudou (diff)
- Filtros no histórico: por usuário, tipo de ação, data
- Versionamento com snapshots completos do documento
- Restaurar versão anterior (salva versão atual antes de restaurar)
- Comparar duas versões (diff)

### WebSocket — Presença em tempo real
- Rooms por documento: cada documento tem sua sala isolada
- Presence tracking: join/leave automático ao conectar/desconectar
- Broadcast de eventos para todos os clientes da sala
- Reconexão automática com backoff (até 5 tentativas)
- Autenticação via subprotocol JWT

### CRDT com Yjs
**Por que CRDT?** Operational Transformation (OT, usado no Google Docs original) resolve conflitos de edição simultânea via transformações matemáticas que precisam de um servidor central como árbitro. CRDT resolve o mesmo problema por design — cada operação é comutativa e idempotente, então dois clientes que aplicam as mesmas operações em qualquer ordem chegam ao mesmo resultado, sem árbitro central.

**Arquitetura:**
```
Cliente A (Yjs doc) ──┐
                      ├── WebSocket ──► Go Backend (relay) ──► PostgreSQL
Cliente B (Yjs doc) ──┘                     │
                                            └── yjs_updates table
```

O backend Go **não precisa entender** o conteúdo Yjs — ele é um relay puro. Isso é intencional: a lógica de merge vive nos clientes.

**Protocolo de sincronização:**
1. Cliente conecta → envia `SyncStep1` (state vector: quais operações já tem)
2. Peer recebe `SyncStep1` → responde com `SyncStep2` (diff: operações que o outro não tem)
3. Cliente recebe `SyncStep2` → aplica ao doc local, marcado como `synced`
4. Qualquer edição local → envia `Update` para todos os peers via relay
5. Reconexão → repete a partir do passo 1

**Persistência:** O backend salva cada update Yjs na tabela `yjs_updates` (bytea). Quando um novo cliente conecta, o hub envia todos os updates salvos para ele reconstruir o estado completo — assim o documento persiste mesmo sem nenhum peer online.

**Rich text com TipTap:**
- O TipTap usa `Y.XmlFragment("content")` como CRDT subjacente (não `Y.Text`)
- `Y.XmlFragment` é uma árvore XML colaborativa — cada nó (parágrafo, heading, bold) é uma operação CRDT independente
- Isso significa que dois usuários podem formatar o mesmo trecho simultaneamente sem conflito
- A extensão `@tiptap/extension-collaboration` conecta o ProseMirror (engine do TipTap) ao Yjs automaticamente

### Awareness visual
- Cursors remotos em tempo real: linha colorida na posição de caret de cada usuário
- Badges com nome aparecem por 2.5s ao mover o cursor
- Seleções coloridas de outros usuários (highlight semi-transparente)
- Tooltips com nome e status "editing" nos avatares
- Dot pulsante no avatar quando o usuário está com cursor ativo

---

## Estrutura do projeto

```
text-editor/
├── client/                     # Next.js frontend
│   └── src/
│       ├── app/
│       │   ├── dashboard/      # Lista de documentos
│       │   ├── editor/[id]/    # Editor principal
│       │   ├── login/
│       │   └── signup/
│       ├── components/
│       │   ├── EditorToolbar.tsx     # Barra de formatação TipTap
│       │   ├── Navbar.tsx
│       │   ├── RemoteCursors.tsx     # Cursors e seleções remotas
│       │   ├── TiptapEditor.tsx      # Wrapper do editor
│       │   └── UserPresence.tsx      # Avatares online
│       ├── hooks/
│       │   ├── useWebSocket.ts       # Conexão WebSocket + presença
│       │   └── useYjsEditor.ts       # Yjs doc + provider + sync state
│       ├── lib/
│       │   ├── api.ts                # Cliente HTTP (axios)
│       │   ├── websocket.ts          # WebSocketClient class
│       │   └── yjs-provider.ts       # Protocolo de sync Yjs
│       └── store/
│           └── authStore.ts          # Zustand (JWT + user)
│
└── server/                     # Go backend
    ├── cmd/server/main.go
    ├── internal/
    │   ├── auth/               # JWT + bcrypt
    │   ├── config/
    │   ├── database/           # Conexão + migrations runner
    │   ├── handlers/           # HTTP handlers (Gin)
    │   ├── middleware/         # Auth middleware
    │   ├── models/             # GORM models
    │   ├── services/           # Lógica de negócio
    │   └── websocket/          # Hub + Client (gorilla/websocket)
    ├── migrations/             # SQL versionado (golang-migrate)
    │   ├── 000001 → documents
    │   ├── 000002 → users
    │   ├── 000003 → user_id em documents
    │   ├── 000004 → document_collaborators
    │   ├── 000005 → document_share_links
    │   ├── 000006 → notifications
    │   ├── 000007 → document_histories
    │   ├── 000008 → document_versions
    │   ├── 000009 → yjs_updates
    │   └── 000010 → content_format em documents
    └── pkg/response/           # Respostas padronizadas
```

---

## Como rodar

### Pré-requisitos
- Docker e docker-compose
- Node.js 18+

### Backend

```bash
cd server

# Copiar variáveis de ambiente
cp .env.example .env

# Subir PostgreSQL + backend (migrations rodam automaticamente)
docker-compose up -d

# Ver logs
docker-compose logs -f backend
```

### Frontend

```bash
cd client

# Instalar dependências
npm install

# Criar .env.local
echo "NEXT_PUBLIC_API_URL=http://localhost:8080" > .env.local
echo "NEXT_PUBLIC_WS_URL=ws://localhost:8080" >> .env.local

# Rodar em desenvolvimento
npm run dev
```

Acesse `http://localhost:3000`.

---

## API — endpoints principais

### Auth
| Método | Rota | Descrição |
|--------|------|-----------|
| POST | `/api/auth/signup` | Criar conta |
| POST | `/api/auth/login` | Login (retorna JWT) |
| GET | `/api/auth/me` | Dados do usuário autenticado |

### Documentos
| Método | Rota | Descrição |
|--------|------|-----------|
| POST | `/api/documents` | Criar documento |
| GET | `/api/documents` | Listar (owned + shared) |
| GET | `/api/documents/:id` | Buscar por ID + permissão |
| PUT | `/api/documents/:id` | Atualizar |
| DELETE | `/api/documents/:id` | Deletar (owner only) |

### Colaboração
| Método | Rota | Descrição |
|--------|------|-----------|
| POST | `/api/documents/:id/collaborators` | Adicionar por email |
| GET | `/api/documents/:id/collaborators` | Listar |
| PUT | `/api/documents/:id/collaborators/:user_id` | Atualizar permissão |
| DELETE | `/api/documents/:id/collaborators/:user_id` | Remover |
| POST | `/api/documents/:id/share-link` | Criar link público |
| DELETE | `/api/documents/:id/share-link` | Remover link |

### Histórico e versões
| Método | Rota | Descrição |
|--------|------|-----------|
| GET | `/api/documents/:id/history` | Histórico com filtros |
| GET | `/api/documents/:id/versions` | Listar versões |
| GET | `/api/documents/:id/versions/:n` | Ver versão específica |
| POST | `/api/documents/:id/versions/:n/restore` | Restaurar versão |
| GET | `/api/documents/:id/versions/compare?v1=N&v2=M` | Comparar versões |

### Notificações
| Método | Rota | Descrição |
|--------|------|-----------|
| GET | `/api/notifications` | Listar + contador de não lidas |
| PUT | `/api/notifications/:id/read` | Marcar como lida |
| PUT | `/api/notifications/read-all` | Marcar todas como lidas |
| DELETE | `/api/notifications/:id` | Deletar |

### WebSocket
| Rota | Descrição |
|------|-----------|
| `WS /ws/documents/:id` | Conectar ao documento (auth via subprotocol) |

---

## Decisões técnicas relevantes

**Por que Go no backend?** Performance nativa para WebSockets com muitas conexões simultâneas. Goroutines são muito mais leves que threads — cada conexão WebSocket roda em sua própria goroutine.

**Por que Yjs e não implementar CRDT próprio?** Yjs é battle-tested, usado em produção por Notion, Figma, e outros. Implementar CRDT do zero (LSEQ, RGA, Logoot) é um projeto de dissertação. O aprendizado está em *entender e integrar* o Yjs, não em reimplementar algoritmos já provados.

**Por que o backend Go não precisa entender Yjs?** O CRDT é peer-to-peer por natureza. O servidor é só um relay — repassa bytes de um cliente para os outros. Isso torna o backend stateless em relação ao conteúdo, e fácil de escalar horizontalmente (com um broker de mensagens como Redis Pub/Sub entre instâncias).

**Migrations versionadas vs AutoMigrate:** AutoMigrate do GORM é conveniente mas sem controle de versão, sem rollback, e não recomendado para produção. `golang-migrate` com arquivos SQL numerados dá rastreabilidade total — cada mudança no schema é um arquivo `up.sql` + `down.sql` commitado no repositório.

**Autenticação WebSocket via subprotocol:** Browsers não permitem headers customizados em conexões WebSocket. A solução comum (e insegura) é passar o token como query param — ele aparece em logs de servidor, logs de proxy, histórico do browser. O subprotocol (`Sec-WebSocket-Protocol`) é um header WebSocket padrão que passa no handshake sem aparecer na URL.

---

## Próximos passos

- [ ] Offline support com `y-indexeddb` (editar sem internet, sync ao reconectar)
- [ ] Compactação de updates Yjs antigos (`Y.mergeUpdates`)
- [ ] Deploy público (Railway/Render para o backend, Vercel para o frontend)