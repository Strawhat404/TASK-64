# Local Operations & Compliance Console

A high-security, offline-first full-stack system for internal network deployment. Manages service scheduling, financial reconciliation, content governance, and compliance workflows with multi-tenant support and role-based access control.

## Prerequisites

- Docker Engine 24+ and Docker Compose v2
- (Optional for local dev) Go 1.22+, Node.js 20+, PostgreSQL 16+

## Quick Start

```bash
cd repo/
cp .env.example .env    # Review and adjust as needed
docker compose up -d    # Builds and starts all services
```

Access the console at **https://localhost:3443** (self-signed TLS certificate).

Default credentials:
- Username: `admin`
- Password: `Admin12345!!!`

## Architecture

```
Client (Browser)
   |
   | HTTPS :3443
   v
+--------+     +----------+     +-----------+
| Caddy  |---->| Frontend |     | PostgreSQL|
| (TLS)  |     | (React)  |     | (Data)    |
|        |---->| Backend  |---->|           |
+--------+     | (Go/Echo)|     +-----------+
               +----------+
```

**Four-container Docker Compose stack:**
- **caddy** — Reverse proxy with self-signed TLS, security headers (HSTS, X-Frame-Options, CSP)
- **backend** — Go/Echo API server on :8080 (internal only)
- **frontend** — React SPA served via nginx on :80 (internal only)
- **db** — PostgreSQL 16 with initialization scripts

Only Caddy's port 3443 is exposed externally. All other services communicate via the internal `compliance-net` Docker network.

## RBAC Roles

| Role | Access |
|------|--------|
| **Administrator** | Full system access, user management, security controls |
| **Scheduler** | Service catalog, scheduling, staff management |
| **Reviewer** | Content governance, moderation queue |
| **Auditor** | Audit logs, reconciliation, exception management |

## API Route Groups

| Group | Prefix | Description |
|-------|--------|-------------|
| Auth | `/api/auth` | Login, logout, session |
| Users | `/api/users` | User CRUD (admin) |
| Services | `/api/services` | Service catalog with pricing |
| Schedules | `/api/schedules` | Scheduling with conflict detection |
| Staff | `/api/staff` | Staff roster management |
| Audit | `/api/audit/logs` | Audit log viewer |
| Governance | `/api/governance` | Content moderation, gray-release, versioning |
| Reconciliation | `/api/reconciliation` | Financial matching, exceptions |
| Security | `/api/security` | Encryption, audit ledger, legal holds |

## Security Features

- **Authentication**: Argon2id password hashing, 12-character minimum passwords
- **Session Management**: HttpOnly + Secure + SameSite=Strict cookies, 30-minute idle timeout
- **Brute Force Protection**: CAPTCHA after 3 failures, account lockout after 5 failures within 15-minute window (30-minute lockout)
- **Rate Limiting**: 300 requests/min per user and per IP; 10 requests/min on login endpoint
- **Encryption at Rest**: AES-256-GCM for sensitive fields (bank accounts, tax IDs) with quarterly key rotation
- **Audit Ledger**: SHA-256 hash-chained immutable log with DB-level triggers preventing modification
- **TLS Certificate Auto-Detection**: Backend auto-detects TLS certificates in `backend/certs/` and enables HTTPS when present
- **Legal Holds**: Prevent data deletion during litigation/compliance holds
- **Data Retention**: 7-year default retention with configurable policies and secure deletion workflow

## Configuration

Key environment variables (see `.env.example` for full list):

| Variable | Default | Description |
|----------|---------|-------------|
| `POSTGRES_USER` | `compliance` | Database user |
| `POSTGRES_PASSWORD` | `compliance_secret` | Database password |
| `POSTGRES_DB` | `compliance_console` | Database name |
| `MASTER_KEY` | (dev default) | AES-256 master encryption key (set in production!) |

## Testing

```bash
# Run all tests (unit + API/integration) — requires running stack
./run_tests.sh

# Health check
./tests/health_check.sh

# API integration tests (individually)
./API_tests/api_test.sh
./API_tests/authz_boundary_test.sh

# Unit tests (individually, requires Go 1.22+)
cd unit_tests && go test -v ./...
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| DB connection refused | Ensure `db` container is healthy: `docker compose ps` |
| TLS certificate warning | Expected with self-signed cert; add exception in browser |
| Migration errors | Check PostgreSQL logs: `docker compose logs db` |
| Master key not set | Set `MASTER_KEY` env var; dev default logs a warning |
| Port 3443 in use | Change Caddy port in `docker-compose.yml` and `Caddyfile` |

## Project Structure

```
repo/
├── backend/
│   ├── cmd/main.go              # Entry point, route registration
│   ├── certs/                   # TLS certificates (auto-detected)
│   ├── db/                      # SQL schema and migrations
│   │   └── 06_enhancements.sql  # Relationships, backup staff, credentials, availability
│   ├── internal/
│   │   ├── handlers/            # HTTP request handlers
│   │   ├── middleware/          # Auth, rate limiting, security
│   │   ├── models/              # Domain structs
│   │   └── services/            # Business logic
│   ├── Dockerfile
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── components/          # Sidebar, ProtectedRoute, etc.
│   │   ├── context/             # AuthContext
│   │   └── pages/               # All page components
│   ├── Dockerfile
│   └── package.json
├── API_tests/                   # Integration test scripts
├── unit_tests/                  # Go unit tests
├── tests/
│   └── health_check.sh          # Health check script
├── run_tests.sh                 # Unified test runner (Docker-based)
├── Caddyfile                    # Reverse proxy config
└── docker-compose.yml           # Orchestration
```
