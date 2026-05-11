# Docs Editor — Editor Colaborativo em Tempo Real

Editor de documentos colaborativo inspirado no Google Docs, construído do zero com foco em aprender e demonstrar tecnologias reais de sincronização distribuída (**CRDT via Yjs**) e comunicação em tempo real (**WebSockets**).

> 📖 [Read in English](./README.md)

---

## Stack

| Camada | Tecnologia |
|--------|-----------|
| Frontend | Next.js 16, TypeScript, Tailwind CSS v4 |
| Editor | TipTap v2 + extensões de colaboração |
| CRDT | Yjs + y-protocols |
| Tempo real | WebSocket (gorilla/websocket) |
| Backend | Go 1.22 + Gin |
| Banco | PostgreSQL + golang-migrate |
| Infra | Docker + docker-compose |

---

## Funcionalidades implementadas

### Autenticação
- Registro e login com JWT (access token 24h, refresh token 30 dias via cookie HttpOnly)
- Senhas com bcrypt
- Middleware de proteção de rotas (HTTP e WebSocket)
- Autenticação WebSocket via **subprotocol** (`Sec-WebSocket-Protocol`) — mais seguro que query params, token não aparece em logs
- Verificação de expiração do JWT na reidratação do store — sessões expiradas são limpas automaticamente ao carregar a página
- **Refresh token:** interceptor do `axios` reenvia requisições que falharam com 401 após chamar `POST /api/auth/refresh`; o cliente WebSocket trata o código de fechamento `4001` (token expirado em conexão de longa duração) e renova sem interromper a sessão
- **Rate limiting nos endpoints de auth:** login limitado a 5 req/min por IP, signup a 3 req/min por IP (token bucket em memória com limpeza automática)
- `PATCH /api/auth/me` — atualizar nome de exibição

### Documentos
- CRUD completo com owner automático
- Campo `content_format` para distinguir texto legado (`"text"`) de rich text (`"tiptap"`)
- Preload de relação `User` em todas as respostas
- **Soft delete:** `DELETE /api/documents/:id` define `deleted_at` em vez de remover o registro; documentos ficam recuperáveis na Lixeira por 30 dias antes de serem excluídos permanentemente
- **Lixeira e restauração:** `GET /api/documents/trash` lista documentos excluídos; `POST /api/documents/:id/restore` os recupera
- **Busca full-text:** `GET /api/documents?q=termo` usa `tsvector` + `websearch_to_tsquery` do PostgreSQL com índice GIN; resultados são ordenados por `ts_rank` (título tem peso maior que conteúdo)
- **Paginação:** `?page=N&limit=N` na listagem de documentos

### Sistema de permissões
- Três níveis: `owner`, `editor`, `viewer`
- Owner: todas as operações
- Editor: pode editar, não pode deletar nem gerenciar colaboradores
- Viewer: somente leitura

### Colaboração
- Adicionar colaboradores por email com seletor de permissão (editor / viewer)
- Atualizar permissão de colaboradores inline
- Remover colaboradores
- Links públicos de compartilhamento com permissão configurável e expiração opcional

### Notificações
- Ao ser adicionado como colaborador
- Ao ter permissão alterada
- Ao colaborador editar o documento (owner recebe)
- Marcar como lida individualmente ou em lote
- Badge de não lidas na navbar
- Dropdown em tempo real acessível de qualquer página
- **Push em tempo real via WebSocket:** notificações são entregues instantaneamente pela conexão WS existente (mensagem `notification:new`) sem polling — o `Hub` mantém um índice `UserClients` para alcançar todas as conexões ativas de um usuário específico

### Histórico e versionamento
- Histórico de edições: quem editou, quando e o que mudou (diff)
- Filtros no histórico: por usuário, tipo de ação, data
- Versionamento com snapshots completos do documento
- Restaurar versão anterior (salva versão atual antes de restaurar)
- Comparar duas versões (diff)
- Painel de histórico no editor (últimas 20 versões, slide-in pela direita)

### WebSocket — Presença em tempo real
- Rooms por documento: cada documento tem sua sala isolada
- Presence tracking: join/leave automático ao conectar/desconectar
- Broadcast de eventos para todos os clientes da sala
- Reconexão automática com backoff exponencial (até 5 tentativas)
- Autenticação via subprotocol JWT
- **Verificação de expiração do JWT em conexões de longa duração:** um ticker de 15 min no `WritePump` usa `ParseUnverified` para ler o claim `exp` e envia código de fechamento `4001` se expirado; o cliente intercepta `4001`, chama `POST /api/auth/refresh` e reconecta automaticamente

### CRDT com Yjs
**Por que CRDT?** Operational Transformation (OT, usado no Google Docs original) resolve conflitos de edição simultânea via transformações matemáticas que precisam de um servidor central como árbitro. CRDT resolve o mesmo problema por design — cada operação é comutativa e idempotente, então dois clientes que aplicam as mesmas operações em qualquer ordem chegam ao mesmo resultado, sem árbitro central.

**Arquitetura:**
```
Cliente A (Yjs doc) ──┐
                       ├── WebSocket ──► Go Backend (relay) ──► PostgreSQL
Cliente B (Yjs doc) ──┘                     │
                                            ├── yjs_updates table
                                            └── yjs_snapshots table
```

O backend Go **não precisa entender** o conteúdo Yjs — ele é um relay puro. Isso é intencional: a lógica de merge vive nos clientes.

**Protocolo de sincronização:**
1. Cliente conecta → envia `SyncStep1` (state vector: quais operações já tem)
2. Peer recebe `SyncStep1` → responde com `SyncStep2` (diff: operações que o outro não tem)
3. Cliente recebe `SyncStep2` → aplica ao doc local, marcado como `synced`
4. Qualquer edição local → envia `Update` para todos os peers via relay
5. Reconexão → repete a partir do passo 1

**Persistência:** O backend salva cada update Yjs na tabela `yjs_updates` (bytea). Quando um novo cliente conecta, o hub envia todos os updates salvos para ele reconstruir o estado completo — assim o documento persiste mesmo sem nenhum peer online.

**Compactação:** Um worker Node.js (`compactor/`) executa `Y.mergeUpdates` para fundir updates individuais em um único snapshot quando um threshold de contagem ou tamanho é atingido. A compactação também roda quando o último peer desconecta, mantendo `yjs_updates` sem crescimento ilimitado.

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

### UI e Design System
- **Dark / Light mode** com toggle persistente (localStorage via Zustand)
- Theming baseado em variáveis CSS — todos os componentes usam `var(--bg-base)`, `var(--text-primary)`, etc.
- Fonte **Inter** (Google Fonts via `next/font`)
- Skeleton loading no dashboard de documentos
- Páginas de login e signup em duas colunas (painel escuro de branding + painel de formulário)
- Sino de notificações com badge de não lidas na navbar
- Página de perfil (`/profile`) com edição inline de nome e logout
- Toasts de feedback para todas as ações assíncronas (sucesso / erro)
- Painel de histórico de versões com modal de confirmação antes de restaurar
- Modal de colaboradores com convite por email, seletor de permissão, gerenciamento de link público
- **Barra de busca no dashboard** com debounce de 350 ms — filtra documentos por título ou conteúdo via FTS
- **Painel Lixeira** no dashboard — exibe documentos excluídos com botão de restaurar; purga permanente após 30 dias
- **Botão de soft delete** em cada card de documento (aparece ao passar o mouse, clique único, sem confirmação por ser reversível)

---

## Estrutura do projeto

```
text-editor/
├── client/                     # Next.js frontend
│   └── src/
│       ├── app/
│       │   ├── dashboard/      # Lista de documentos com skeleton loading
│       │   ├── editor/[id]/    # Editor principal
│       │   ├── login/          # Layout duas colunas
│       │   ├── signup/         # Layout duas colunas
│       │   └── profile/        # Página de perfil do usuário
│       ├── components/
│       │   ├── editor/
│       │   │   ├── EditorToolbar.tsx       # Barra de formatação TipTap
│       │   │   ├── VersionHistory.tsx      # Painel de histórico slide-in
│       │   │   └── CollaboratorsModal.tsx  # Modal de compartilhamento
│       │   ├── layout/
│       │   │   ├── Navbar.tsx              # Com toggle de tema, notificações, avatar
│       │   │   └── ThemeProvider.tsx       # Aplica .dark no <html>
│       │   ├── notifications/
│       │   │   └── NotificationBell.tsx    # Dropdown com badge de não lidas
│       │   ├── presence/
│       │   │   ├── RemoteCursors.tsx       # Cursors e seleções remotas
│       │   │   └── UserPresence.tsx        # Avatares online
│       │   └── ui/
│       │       ├── Button.tsx
│       │       ├── Input.tsx               # Theme-aware, com toggle de senha
│       │       ├── Modal.tsx
│       │       ├── Skeleton.tsx            # DocumentCardSkeleton
│       │       └── Toast.tsx
│       ├── hooks/
│       │   ├── useWebSocket.ts             # Conexão WebSocket + presença
│       │   ├── useVersionHistory.ts        # Buscar + restaurar versões
│       │   └── useYjsEditor.ts             # Yjs doc + provider + sync state
│       ├── lib/
│       │   ├── api.ts                      # Todas as chamadas de API
│       │   ├── axios.ts                    # Instância axios + interceptor de auth
│       │   ├── utils.ts                    # Helper cn()
│       │   ├── websocket.ts                # Classe WebSocketClient
│       │   └── yjs-provider.ts             # Protocolo de sync Yjs
│       ├── store/
│       │   ├── authStore.ts                # Zustand (JWT + user + isHydrated)
│       │   ├── themeStore.ts               # Dark/light mode (persistido)
│       │   └── toastStore.ts               # Toasts globais
│       └── types/
│           ├── collaborator.ts
│           ├── document.ts
│           ├── notification.ts
│           ├── user.ts
│           ├── version.ts
│           └── index.ts
│
└── server/                     # Go backend
    ├── cmd/server/main.go      # Entry-point enxuto (~17 linhas): carrega .env, chama app.Run()
    ├── compactor/              # Worker Node.js para operações de merge Yjs
    │   └── index.js            # Endpoints /compact, /state-vector, /health
    ├── internal/
    │   ├── app/                # Wiring de dependências + bootstrap do servidor (app.Run)
    │   ├── auth/               # JWT + bcrypt
    │   ├── config/             # Config struct + leitura de env vars (incl. ALLOWED_ORIGINS)
    │   ├── database/           # Conexão + migrations runner
    │   ├── handlers/           # HTTP handlers (Gin)
    │   ├── middleware/         # Auth + rate-limit middleware
    │   ├── models/             # GORM models
    │   ├── router/             # Criação do engine Gin (router.go) + registro de rotas (routes.go)
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
    │   ├── 000010 → content_format em documents
    │   ├── 000011 → lamport_ts + client_id em yjs_updates
    │   ├── 000012 → yjs_snapshots
    │   ├── 000013 → yjs_snapshot em document_versions
    │   ├── 000014 → refresh_token em users
    │   ├── 000015 → soft delete (deleted_at) em documents
    │   └── 000016 → coluna tsvector FTS + índice GIN em documents
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

# (Opcional) Liberar origens adicionais no CORS — separadas por vírgula
# echo "ALLOWED_ORIGINS=http://localhost:3000,https://meuapp.com" >> .env

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
| POST | `/api/auth/login` | Login (retorna access JWT + define cookie de refresh) |
| POST | `/api/auth/refresh` | Renovar access token via cookie HttpOnly |
| GET | `/api/auth/me` | Dados do usuário autenticado |
| PATCH | `/api/auth/me` | Atualizar nome de exibição |
| POST | `/api/auth/logout` | Revogar refresh token |

### Documentos
| Método | Rota | Descrição |
|--------|------|-----------|
| POST | `/api/documents` | Criar documento |
| GET | `/api/documents?q=&page=&limit=` | Listar (owned + shared), com busca FTS |
| GET | `/api/documents/trash` | Listar documentos na lixeira |
| GET | `/api/documents/:id` | Buscar por ID + permissão |
| PUT | `/api/documents/:id` | Atualizar |
| DELETE | `/api/documents/:id` | Soft delete — move para Lixeira |
| POST | `/api/documents/:id/restore` | Restaurar da Lixeira |

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

### CRDT (Yjs)
| Método | Rota | Descrição |
|--------|------|-----------|
| GET | `/api/documents/:id/yjs-updates` | Updates CRDT brutos |
| GET | `/api/documents/:id/yjs-state-vector` | State vector atual (otimização de reconexão) |

---

## Decisões técnicas relevantes

**Por que Go no backend?** Performance nativa para WebSockets com muitas conexões simultâneas. Goroutines são muito mais leves que threads — cada conexão WebSocket roda em sua própria goroutine.

**Por que Yjs e não implementar CRDT próprio?** Yjs é battle-tested, usado em produção por Notion, Figma, e outros. Implementar CRDT do zero (LSEQ, RGA, Logoot) é um projeto de dissertação. O aprendizado está em *entender e integrar* o Yjs, não em reimplementar algoritmos já provados.

**Por que o backend Go não precisa entender Yjs?** O CRDT é peer-to-peer por natureza. O servidor é só um relay — repassa bytes de um cliente para os outros. Isso torna o backend stateless em relação ao conteúdo, e fácil de escalar horizontalmente (com um broker de mensagens como Redis Pub/Sub entre instâncias).

**Migrations versionadas vs AutoMigrate:** AutoMigrate do GORM é conveniente mas sem controle de versão, sem rollback, e não recomendado para produção. `golang-migrate` com arquivos SQL numerados dá rastreabilidade total — cada mudança no schema é um arquivo `up.sql` + `down.sql` commitado no repositório.

**Autenticação WebSocket via subprotocol:** Browsers não permitem headers customizados em conexões WebSocket. A solução comum (e insegura) é passar o token como query param — ele aparece em logs de servidor, logs de proxy, histórico do browser. O subprotocol (`Sec-WebSocket-Protocol`) é um header WebSocket padrão que passa no handshake sem aparecer na URL.

**Compactação CRDT com worker Node.js:** O Go não tem uma implementação nativa de Yjs. Para mesclar updates (`Y.mergeUpdates`) e gerar state vectors, um worker Node.js isolado (`compactor/`) expõe endpoints HTTP consumidos pelo backend Go. Isso evita WebAssembly e mantém cada responsabilidade em sua linguagem natural. O compactor aplica uma política de duplo threshold: compactar quando os updates excedem um limite de contagem **ou** de tamanho total. A compactação idle também roda quando o último peer desconecta.

**Restore de versão via `document-content-reset`:** Restaurar uma versão via snapshot CRDT binário não funciona para time-travel — `Y.applyUpdate` é aditivo, e operações com Lamport clock mais alto sempre vencem. A solução correta é transmitir o conteúdo JSON da versão via WebSocket (`document-content-reset`) e cada cliente chama `editor.commands.setContent()`, que gera novas operações Yjs com o clock máximo atual — sobrescrevendo o conteúdo antigo em todos os peers. O estado CRDT do servidor também é limpo no restore para que clientes que conectem depois partam de um estado limpo.

**Ordering causal com Lamport timestamps:** O backend decodifica o formato binário Yjs v1 em Go (sem instanciar um Y.Doc) para extrair o `LamportTS` e o `ClientID` de cada update. Esses valores são indexados em `yjs_updates` para garantir ordering causal ao reproduzir o histórico para novos peers, e para determinar corretamente o delta a enviar durante reconexão.

**Theming via variáveis CSS:** Em vez de classes utilitárias Tailwind de dark mode espalhadas pelos componentes, todos os tokens de cor são definidos como variáveis CSS em `globals.css` (ex: `--bg-base`, `--text-primary`, `--border`). O componente `ThemeProvider` alterna a classe `.dark` no `<html>`, que troca os valores das variáveis. Isso significa que todo componente é theme-aware por padrão, e adicionar um terceiro tema (ex: alto contraste) requer alterar apenas as definições das variáveis CSS.

**Arquitetura frontend type-safe:** Todos os shapes de resposta da API são definidos em `src/types/` (document, user, collaborator, notification, version). O `authStore` (Zustand + persist) valida a expiração do JWT na reidratação e limpa sessões expiradas antes de qualquer route guard executar — prevenindo estado de auth desatualizado após um longo período inativo.

**Refresh token com SHA-256:** Refresh tokens são strings aleatórias de alta entropia — bcrypt é intencionalmente lento e projetado para senhas de baixa entropia. Os tokens são armazenados como hashes SHA-256 (`hex.EncodeToString(sha256.Sum256(token))`), o que é rápido, resistente a colisões e adequado para segredos pré-validados.

**Notificações em tempo real via WebSocket:** O `Hub` mantém um índice `UserClients map[uuid.UUID]map[*Client]bool` além do índice por documento. O `NotificationService` usa uma interface `NotificationHub` (para evitar importações circulares) e chama `SendToUser` após cada inserção no banco, entregando a notificação instantaneamente em todas as conexões ativas do usuário alvo. O `NotificationBell` no frontend escuta um `CustomEvent` global `notification:new` despachado pelo `yjs-provider.ts`.

**Expiração do JWT em conexões WebSocket de longa duração:** Um JWT de curta duração é adequado para chamadas HTTP (cada requisição revalida), mas uma conexão WebSocket pode ficar aberta por horas. Um `tokenTicker` de 15 minutos no `WritePump` usa `jwt.ParseUnverified` (sem precisar da chave — apenas lê o claim `exp`) e envia o código de fechamento `4001` se expirado. O `WebSocketClient` no frontend captura o `4001`, chama `POST /api/auth/refresh`, atualiza o token no `authStore`, despacha `ws:token-refreshed` e reconecta — sem nenhuma ação do usuário.

**Soft delete e FTS no PostgreSQL:** Documentos nunca são deletados imediatamente — `DELETE /api/documents/:id` define `deleted_at` (campo `DeletedAt` do GORM, filtrado automaticamente de todas as queries). Uma goroutine em background purga registros com mais de 30 dias. A busca full-text usa uma coluna `tsvector` gerada (`GENERATED ALWAYS AS ... STORED`) combinando título (peso A) e conteúdo (peso B), com índice GIN para lookups rápidos com `@@` e ordenação por `ts_rank`. `websearch_to_tsquery` é usado em vez de `plainto_tsquery` para suportar frases entre aspas e exclusões.

**`.dockerignore` no Docker:** O diretório `compactor/` contém um worker Node.js com seu próprio `node_modules`. Sem `.dockerignore`, o `docker build` transfere o `node_modules` para o daemon — causando falhas por nomes de arquivos com caracteres especiais (ex: `@scope/pkg`, `0ecdsa-generate-keypair`). O `.dockerignore` exclui `compactor/node_modules`, `bin/` e arquivos `.env`.

**Clean architecture no servidor Go:** `cmd/server/main.go` é um entry-point enxuto (~17 linhas) que carrega o `.env` e chama `app.Run()`. Todo o wiring de dependências (banco, services, hub WebSocket) vive em `internal/app/app.go`. A criação do engine Gin e a configuração de CORS ficam em `internal/router/router.go`; o registro das rotas é dividido por grupo de domínio (auth, documents, notifications, WebSocket) em `internal/router/routes.go`. Uma struct `Dependencies` substitui a lista crescente de parâmetros de função — adicionar um novo serviço exige apenas atualizar a struct e o `app.go`, sem tocar nas assinaturas das funções de rota. As origens permitidas no CORS são lidas da variável de ambiente `ALLOWED_ORIGINS` (separadas por vírgula), com default `http://localhost:3000`.
