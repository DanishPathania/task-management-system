# Real-Time Collaborative Task Management System

A production-quality, real-time collaborative task management system built from scratch with Go, Next.js 14 (App Router), TypeScript, Tailwind CSS, PostgreSQL, Redis, and WebSockets.

---

## 1. Project Overview

CollabTask is designed for teams that require instant synchronization, low-latency collaboration, and strict data persistence. Team members can organize tasks on a live Kanban board, comment in real time, view live online presence of colleagues, assign tasks, and manage project permissions through project-scoped Role-Based Access Control (RBAC).

---

## 2. Feature List

- **Authentication & Security**:
  - Secure registration, login, and logout.
  - Password hashing with `bcrypt`.
  - JWT token generation, expiration enforcement, and validation.
  - HttpOnly, SameSite cookies with fallback token authorization.
  - Parameterized queries via `pgxpool` preventing SQL injection.
- **Project-Scoped RBAC**:
  - Project **Admin**: create/update/delete projects, manage members & roles, assign tasks, moderate/delete any comments.
  - Project **Member**: view workspace, create/update tasks, change status and priority, post comments, delete own comments, view active presence.
- **Task Management**:
  - Task fields: ID, Project ID, Title, Description, Status (`TODO`, `IN_PROGRESS`, `DONE`), Priority (`LOW`, `MEDIUM`, `HIGH`), Assignee, Due Date, Creator, Timestamps.
  - Live Kanban board with columns and real-time drag/click movements.
  - Filtering by status, priority, and assignee; pagination support.
  - Assignee project membership validation enforced on the backend.
- **Real-Time Collaboration**:
  - Bi-directional WebSockets (`/api/v1/ws`) with project-scoped subscriptions.
  - Real-time event broadcasts (`task.created`, `task.updated`, `task.deleted`, `task.assigned`, `task.status_changed`, `comment.created`, `comment.deleted`, `presence.updated`).
  - Safe concurrent writes with bounded per-client queues.
  - Client heartbeat (ping/pong) and auto-reconnection with exponential backoff.
- **Redis Caching & Online Presence**:
  - Caching read-heavy project metadata with cache invalidation on mutations.
  - Active presence tracking with Redis keys, leases, and auto-cleanup.
  - Horizontal scaling ready: Redis Pub/Sub integration for cross-instance event distribution.
- **DevOps & Infrastructure**:
  - Multi-stage Docker builds for backend and frontend with non-root security.
  - Docker Compose orchestrating PostgreSQL 16, Redis 7, Go API, and Next.js frontend with container health checks and persistent storage volumes.

---

## 3. Technology Stack

| Layer | Technology | Purpose |
|-------|------------|---------|
| **Frontend** | Next.js 14 (App Router), React 18, TypeScript, Tailwind CSS | Responsive, accessible, and fast web application |
| **Backend** | Go 1.22, Chi router, Gorilla WebSocket | High-performance, concurrent REST API & WebSocket server |
| **Database** | PostgreSQL 16 (`pgxpool/v5`) | Normalized, persistent relational data store |
| **Cache & Realtime** | Redis 7 (`go-redis/v9`) | Hot data caching, online presence leases, and cross-instance Pub/Sub |
| **Containerization** | Docker & Docker Compose | Reproducible local development and production orchestration |

---

## 4. System Architecture

```text
  Browser 1 (Admin)                   Browser 2 (Member)
       |                                      |
       |  HTTP REST / WebSocket (TLS)         |  HTTP REST / WebSocket (TLS)
       +-------------------\      /-----------+
                            v    v
                    +--------------------+
                    |   Next.js App      |
                    | (React 18 / Client)|
                    +--------------------+
                               |
                               v
                    +--------------------+
                    |  Go API Server     |
                    | (REST & WS Hub)    |
                    +--------------------+
                      /        |       \
                     /         |        \
                    v          v         v
        +--------------+ +-----------+ +---------------+
        |  PostgreSQL  | |   Redis   | | Redis Pub/Sub |
        |  (Database)  | |  (Cache & | | (Cross-Node   |
        |              | | Presence) | |  Sync)        |
        +--------------+ +-----------+ +---------------+
```

---

## 5. Repository Structure

```text
Task_management/
├── backend/
│   ├── cmd/
│   │   └── api/
│   │       └── main.go              # Server entry point & route definitions
│   ├── internal/
│   │   ├── auth/                    # Hashing (bcrypt) & JWT claims
│   │   ├── config/                  # Environment loader & validation
│   │   ├── database/                # pgxpool connection & migration runner
│   │   ├── handler/                 # HTTP REST & WebSocket handlers
│   │   ├── middleware/              # Auth, CORS, logging, recovery
│   │   ├── models/                  # Domain entities & event envelopes
│   │   ├── redis/                   # Cache, presence leases, pub/sub
│   │   ├── repository/              # PostgreSQL data access layer
│   │   ├── service/                 # Business logic & authorization rules
│   │   ├── validation/              # Input & parameter validation
│   │   └── websocket/               # Hub, Client, write/read pumps
│   ├── migrations/                  # Versioned PostgreSQL DDL migrations
│   ├── tests/                       # Unit tests (auth, validation)
│   ├── Dockerfile                   # Multi-stage Go production image
│   ├── go.mod                       # Dependencies definition
│   └── .env.example                 # Backend environment template
├── frontend/
│   ├── src/
│   │   ├── app/
│   │   │   ├── (auth)/
│   │   │   │   ├── login/page.tsx   # Login page
│   │   │   │   └── register/page.tsx# Registration page
│   │   │   ├── dashboard/page.tsx   # Projects dashboard
│   │   │   ├── projects/[projectId]/# Real-time workspace Kanban board
│   │   │   ├── layout.tsx           # Root layout with AuthProvider
│   │   │   └── page.tsx             # Landing hero page
│   │   ├── components/              # Modals, TaskCards, Navbar
│   │   ├── hooks/                   # useAuth and application state
│   │   ├── lib/                     # api.ts (REST) & websocket.ts (real-time)
│   │   ├── styles/globals.css       # Tailwind CSS & theme variables
│   │   └── types/index.ts           # Strict TypeScript contracts
│   ├── Dockerfile                   # Multi-stage Next.js production image
│   ├── package.json
│   ├── tsconfig.json
│   └── .env.example                 # Frontend environment template
├── docker-compose.yml               # Multi-container orchestration
├── .gitignore
├── .dockerignore
└── README.md
```

---

## 6. Prerequisites

To run this application, make sure you have installed:
- **Docker** and **Docker Compose** (recommended for running all 4 services)
- Or for running without Docker:
  - **Go 1.22+**
  - **Node.js 20+** and **npm**
  - **PostgreSQL 16**
  - **Redis 7**

---

## 7. Quickstart with Docker Compose

1. **Clone repository and configure environment**:
   ```bash
   cp backend/.env.example backend/.env
   cp frontend/.env.example frontend/.env.local
   ```

2. **Launch all services**:
   ```bash
   docker compose up --build
   ```
   Docker Compose will:
   - Start PostgreSQL with health check on port `5432`.
   - Start Redis with AOF persistence on port `6379`.
   - Run the Go backend API on port `8080`, applying schema migrations automatically on startup.
   - Run the Next.js frontend on port `3000`.

3. **Open the application**:
   - Access the Web UI: `http://localhost:3000`
   - Access the REST API: `http://localhost:8080/api/v1/health`

---

## 8. Local Development (Running Services Manually)

### Step 1: Start PostgreSQL and Redis
Ensure PostgreSQL and Redis are running locally on standard ports:
- PostgreSQL: `localhost:5432` with database `taskdb`
- Redis: `localhost:6379`

### Step 2: Start the Go Backend
```bash
cd backend
go mod download
go run cmd/api/main.go
```
The server will start on `:8080` and automatically verify and apply database migrations.

### Step 3: Start the Next.js Frontend
```bash
cd frontend
npm install
npm run dev
```
Open `http://localhost:3000` in your browser.

---

## 9. API Reference

All REST endpoints are rooted at `/api/v1`.

### Authentication
- `POST /auth/register` — Create a new user (`name`, `email`, `password`)
- `POST /auth/login` — Sign in with `email` and `password` (sets `task_jwt` HttpOnly cookie and returns token)
- `POST /auth/logout` — Clears authentication cookie
- `GET /auth/me` — Retrieve profile of the currently authenticated user

### Projects
- `POST /projects` — Create a new project (creator is designated Admin)
- `GET /projects` — List all projects accessible to the authenticated user
- `GET /projects/{projectId}` — Get details of a project
- `PATCH /projects/{projectId}` — Update project name or description (Admin only)
- `DELETE /projects/{projectId}` — Delete project and all associated tasks/comments (Admin only)
- `GET /projects/{projectId}/presence` — List online members currently connected

### Project Members
- `GET /projects/{projectId}/members` — List members and their roles
- `POST /projects/{projectId}/members` — Add an existing user by UUID as a member or admin (Admin only)
- `PATCH /projects/{projectId}/members/{userId}` — Change member role (`admin` or `member`) (Admin only)
- `DELETE /projects/{projectId}/members/{userId}` — Remove a member from the project (Admin only)

### Tasks
- `POST /projects/{projectId}/tasks` — Create a task within a project (`title`, `description`, `status`, `priority`, `assigned_to`, `due_date`)
- `GET /projects/{projectId}/tasks` — List tasks with filters (`status`, `priority`, `assignee`, `page`, `limit`)
- `GET /projects/{projectId}/tasks/{taskId}` — Get details of a specific task
- `PATCH /projects/{projectId}/tasks/{taskId}` — Update task fields (status, priority, assignee, due date, title, description)
- `DELETE /projects/{projectId}/tasks/{taskId}` — Delete a task (Admin or task creator only)

### Comments
- `POST /projects/{projectId}/tasks/{taskId}/comments` — Add a comment to a task
- `GET /projects/{projectId}/tasks/{taskId}/comments` — List comments for a task
- `DELETE /projects/{projectId}/tasks/{taskId}/comments/{commentId}` — Delete a comment (Author or Admin only)

### WebSocket
- `GET /ws` — Upgrade connection to WebSocket with token authentication

---

## 10. WebSocket Protocol & Real-Time Events

### Connection & Authentication
Clients connect via:
```text
ws://localhost:8080/api/v1/ws?token=<JWT_TOKEN>
```
or via the `task_jwt` cookie during the HTTP upgrade handshake.

### Client Messages
To subscribe to project events:
```json
{
  "action": "subscribe",
  "projectId": "11111111-2222-3333-4444-555555555555"
}
```

To unsubscribe:
```json
{
  "action": "unsubscribe",
  "projectId": "11111111-2222-3333-4444-555555555555"
}
```

### Server Event Envelope
```json
{
  "type": "event",
  "event": "task.created",
  "eventId": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "projectId": "11111111-2222-3333-4444-555555555555",
  "timestamp": "2026-10-22T00:00:00Z",
  "payload": { ... }
}
```

Supported Events:
- `task.created`
- `task.updated`
- `task.status_changed`
- `task.assigned`
- `task.deleted`
- `comment.created`
- `comment.deleted`
- `project.member_added`
- `project.member_removed`
- `project.role_changed`
- `presence.updated`

---

## 11. Testing

### Run Backend Unit Tests
```bash
cd backend
go test -v ./tests/...
```

Tests verify:
- Password hashing and bcrypt verification.
- JWT creation, validation, tampering rejection, and expiration handling.
- Input validation rules (email parsing, min password length, task titles, roles).

---

## 12. Backup & Restore (PostgreSQL)

### Creating a Database Backup
```bash
docker exec -t taskmgr_postgres pg_dump -U taskuser -d taskdb -F c -b -v -f /tmp/taskdb_backup.dump
docker cp taskmgr_postgres:/tmp/taskdb_backup.dump ./taskdb_backup.dump
```

### Restoring from Backup
```bash
docker cp ./taskdb_backup.dump taskmgr_postgres:/tmp/taskdb_backup.dump
docker exec -t taskmgr_postgres pg_restore -U taskuser -d taskdb -v --clean /tmp/taskdb_backup.dump
```

---

## 13. Production Deployment Notes

1. **Environment Variables**:
   - Set a strong, randomly generated `JWT_SECRET` (minimum 32 bytes).
   - Set `SERVER_ENV=production`.
   - Set `COOKIE_SECURE=true` and use HTTPS.
   - Configure `CORS_ALLOWED_ORIGINS` to match your production domain.
2. **Reverse Proxy (Nginx / Caddy / Cloudflare)**:
   - Ensure WebSocket upgrades are enabled:
     ```nginx
     location /api/v1/ws {
         proxy_pass http://backend:8080;
         proxy_http_version 1.1;
         proxy_set_header Upgrade $http_upgrade;
         proxy_set_header Connection "upgrade";
         proxy_read_timeout 3600s;
     }
     ```
3. **Multi-Instance Scaling**:
   - The Go backend is stateless. When deploying multiple backend containers behind a load balancer, instances communicate through the configured Redis Pub/Sub channel to synchronize events across instances automatically.
