# Local Operations & Compliance Console

**Project type:** Fullstack (Go + React)

A high-security, offline-first full-stack system for internal network deployment. Manages service scheduling, financial reconciliation, content governance, and compliance workflows with multi-tenant support and role-based access control.

## Prerequisites

- Docker Engine 24+ and Docker Compose v2

## Quick Start

```bash
cd repo/
docker-compose up
```

Access the console at **https://localhost:3443** (accept the self-signed TLS certificate warning in your browser).

## Demo Credentials

All demo users share the same password: `Admin12345!!!`

| Username | Password | Role | Access |
|----------|----------|------|--------|
| `admin` | `Admin12345!!!` | Administrator | Full system access, user management, security controls |
| `scheduler_user` | `Admin12345!!!` | Scheduler | Service catalog, scheduling, staff management |
| `reviewer_user` | `Admin12345!!!` | Reviewer | Content governance, moderation queue |
| `auditor_user` | `Admin12345!!!` | Auditor | Audit logs, reconciliation, exception management |

> These accounts are seeded automatically by the database initialization scripts on first startup.

## Verification

After `docker-compose up` finishes (all four containers healthy), verify the system works:

```bash
# 1. Health check — should return {"status":"ok"}
curl -sk https://localhost:3443/api/health

# 2. Login as admin — should return 200 with user JSON
curl -sk -X POST https://localhost:3443/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"Admin12345!!!"}' \
  -c /tmp/session.txt

# 3. List services (authenticated) — should return 200 with JSON array
curl -sk https://localhost:3443/api/services -b /tmp/session.txt

# 4. Check audit logs — should return 200 with paginated entries
curl -sk https://localhost:3443/api/audit/logs -b /tmp/session.txt
```

**Expected browser flow:**
1. Open `https://localhost:3443` and accept the certificate warning.
2. Login with `admin` / `Admin12345!!!` — you should see the Dashboard with schedule stats and quick actions.
3. Navigate to **Services** — verify the service catalog table loads.
4. Navigate to **Schedules** — verify date picker and schedule list renders.
5. Navigate to **Security** — verify encryption keys and audit ledger sections load.

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

## API Endpoints

| Method | Endpoint | Roles | Description |
|--------|----------|-------|-------------|
| GET | `/api/health` | Public | Health check |
| POST | `/api/auth/login` | Public | Authenticate user |
| POST | `/api/auth/logout` | Authenticated | End session |
| GET | `/api/auth/session` | Authenticated | Validate session |
| GET | `/api/users` | Administrator | List users |
| GET | `/api/users/:id` | Administrator | Get user |
| POST | `/api/users` | Administrator | Create user |
| PUT | `/api/users/:id` | Administrator | Update user |
| DELETE | `/api/users/:id` | Administrator | Deactivate user |
| GET | `/api/services` | Authenticated | List services |
| GET | `/api/services/:id` | Authenticated | Get service |
| GET | `/api/services/:id/pricing` | Authenticated | Calculate pricing |
| POST | `/api/services` | Administrator, Scheduler | Create service |
| PUT | `/api/services/:id` | Administrator, Scheduler | Update service |
| DELETE | `/api/services/:id` | Administrator | Deactivate service |
| GET | `/api/schedules` | Authenticated | List schedules |
| POST | `/api/schedules` | Administrator, Scheduler | Create schedule |
| PUT | `/api/schedules/:id` | Administrator, Scheduler | Update schedule |
| DELETE | `/api/schedules/:id` | Administrator, Scheduler | Cancel schedule |
| POST | `/api/schedules/:id/confirm` | Authenticated | Confirm assignment |
| GET | `/api/schedules/available-staff` | Authenticated | Find available staff |
| POST | `/api/schedules/:id/backup` | Administrator, Scheduler | Request backup staff |
| POST | `/api/schedules/backup/:id/confirm` | Administrator, Scheduler | Confirm backup |
| GET | `/api/staff` | Authenticated | List staff |
| GET | `/api/staff/:id` | Authenticated | Get staff member |
| POST | `/api/staff` | Administrator, Scheduler | Create staff |
| PUT | `/api/staff/:id` | Administrator, Scheduler | Update staff |
| DELETE | `/api/staff/:id` | Administrator | Delete staff |
| GET | `/api/staff/:id/credentials` | Authenticated | List credentials |
| POST | `/api/staff/:id/credentials` | Administrator, Scheduler | Add credential |
| GET | `/api/staff/:id/availability` | Authenticated | List availability |
| POST | `/api/staff/:id/availability` | Administrator, Scheduler | Add availability |
| GET | `/api/audit/logs` | Auditor, Administrator | Audit log viewer |
| POST | `/api/governance/content` | Administrator, Reviewer | Create content |
| GET | `/api/governance/content` | Authenticated | List content |
| GET | `/api/governance/content/:id` | Authenticated | Get content |
| PUT | `/api/governance/content/:id` | Administrator, Reviewer | Update content |
| POST | `/api/governance/content/:id/submit` | Administrator, Reviewer | Submit for review |
| POST | `/api/governance/content/:id/promote` | Administrator | Promote to published |
| GET | `/api/governance/content/:id/versions` | Authenticated | Version history |
| POST | `/api/governance/content/:id/rollback` | Administrator | Rollback version |
| POST | `/api/governance/content/:id/re-review` | Administrator, Reviewer | Trigger re-review |
| GET | `/api/governance/reviews/pending` | Administrator, Reviewer | Pending reviews |
| POST | `/api/governance/reviews/:id/decide` | Administrator, Reviewer | Review decision |
| GET | `/api/governance/gray-release` | Authenticated | Gray-release items |
| GET | `/api/governance/rules` | Administrator | List rules |
| POST | `/api/governance/rules` | Administrator | Create rule |
| PUT | `/api/governance/rules/:id` | Administrator | Update rule |
| DELETE | `/api/governance/rules/:id` | Administrator | Delete rule |
| GET | `/api/governance/content/:id/versions/diff` | Administrator, Reviewer | Diff versions |
| POST | `/api/governance/relationships` | Administrator, Reviewer | Create relationship |
| GET | `/api/governance/relationships` | Authenticated | List relationships |
| DELETE | `/api/governance/relationships/:id` | Administrator | Delete relationship |
| POST | `/api/reconciliation/import` | Administrator, Auditor | Import feed |
| GET | `/api/reconciliation/feeds` | Administrator, Auditor | List feeds |
| GET | `/api/reconciliation/feeds/:id` | Administrator, Auditor | Get feed |
| POST | `/api/reconciliation/feeds/:id/match` | Administrator, Auditor | Run matching |
| GET | `/api/reconciliation/matches` | Administrator, Auditor | Match results |
| GET | `/api/reconciliation/exceptions` | Administrator, Auditor | List exceptions |
| GET | `/api/reconciliation/exceptions/export` | Administrator, Auditor | Export exceptions |
| GET | `/api/reconciliation/exceptions/:id` | Administrator, Auditor | Get exception |
| PUT | `/api/reconciliation/exceptions/:id/assign` | Administrator, Auditor | Assign exception |
| PUT | `/api/reconciliation/exceptions/:id/resolve` | Administrator, Auditor | Resolve exception |
| GET | `/api/reconciliation/summary` | Administrator, Auditor | Summary stats |
| POST | `/api/security/sensitive` | Administrator | Store sensitive data |
| GET | `/api/security/sensitive` | Administrator | List sensitive data |
| GET | `/api/security/sensitive/:id` | Administrator | Get sensitive data |
| POST | `/api/security/sensitive/:id/reveal` | Administrator | Reveal value |
| DELETE | `/api/security/sensitive/:id` | Administrator | Delete sensitive data |
| POST | `/api/security/keys/rotate` | Administrator | Rotate key |
| GET | `/api/security/keys` | Administrator | Key status |
| GET | `/api/security/keys/rotation-due` | Administrator | Rotation schedule |
| GET | `/api/security/audit-ledger` | Administrator | Audit ledger |
| POST | `/api/security/audit-ledger/verify` | Administrator | Verify chain |
| GET | `/api/security/retention` | Administrator | Retention policies |
| POST | `/api/security/retention/cleanup` | Administrator | Run cleanup |
| GET | `/api/security/rate-limits` | Administrator | Rate limit status |
| POST | `/api/security/legal-holds` | Administrator | Create legal hold |
| GET | `/api/security/legal-holds` | Administrator | List legal holds |
| PUT | `/api/security/legal-holds/:id/release` | Administrator | Release hold |

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

All tests run inside Docker containers. No local Go or Node.js installation required.

```bash
# Run the full test suite (unit + API integration) — requires running stack
docker-compose up -d
./run_tests.sh
```

The test runner (`run_tests.sh`) executes:
1. **Go unit tests** inside a `golang:1.22-alpine` container (tests business logic, encryption, scheduling, reconciliation matching)
2. **API integration tests** via shell scripts inside a container connected to the backend network (real HTTP requests against the live backend)
3. **Authorization boundary tests** validating RBAC enforcement, tenant isolation, and unauthenticated access rejection

## Troubleshooting

| Issue | Solution |
|-------|----------|
| DB connection refused | Ensure `db` container is healthy: `docker-compose ps` |
| TLS certificate warning | Expected with self-signed cert; add exception in browser |
| Migration errors | Check PostgreSQL logs: `docker-compose logs db` |
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
├── API_tests/                   # Integration test scripts (curl-based)
├── unit_tests/                  # Go unit tests (run inside Docker)
├── tests/
│   └── health_check.sh          # Health check script
├── run_tests.sh                 # Unified test runner (Docker-based)
├── Caddyfile                    # Reverse proxy config
└── docker-compose.yml           # Orchestration
```
