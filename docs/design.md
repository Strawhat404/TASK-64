# Local Operations & Compliance Console -- Design Document

**Version:** 1.0.0
**Date:** 2026-04-09
**Status:** Draft

---

## Table of Contents

1. [Overview](#1-overview)
2. [System Architecture](#2-system-architecture)
3. [Security Model](#3-security-model)
4. [Identity and Access Management](#4-identity-and-access-management)
5. [Service Catalog](#5-service-catalog)
6. [Scheduling Engine](#6-scheduling-engine)
7. [Audit Logging](#7-audit-logging)
8. [Deployment Model](#8-deployment-model)
9. [Database Schema Overview](#9-database-schema-overview)

---

## 1. Overview

The Local Operations & Compliance Console is a high-security, offline-first full-stack application designed for deployment on internal networks that may have limited or no internet connectivity. It provides service catalog management, staff scheduling with conflict detection, role-based user administration, and comprehensive audit logging suitable for regulated environments.

### Design Principles

- **Offline-first:** The system must be fully functional without any external network dependency. All assets, dependencies, and runtime components are bundled within the deployment artifacts.
- **Defense in depth:** Multiple layers of security controls (transport encryption, strong authentication, role-based authorization, audit trails) protect against both external and insider threats.
- **Auditability:** Every state-changing operation is recorded in an immutable audit log to support compliance reviews and forensic investigation.
- **Simplicity of deployment:** A single `docker compose up` command brings the entire stack online with no external service dependencies.

---

## 2. System Architecture

### Component Diagram

```
                      +------------------+
                      |   Browser (SPA)  |
                      |   React Client   |
                      +--------+---------+
                               |
                          HTTPS :3443
                               |
                      +--------+---------+
                      |      Caddy       |
                      |  Reverse Proxy   |
                      |  (TLS Termination)|
                      +--------+---------+
                               |
                        HTTP :8080 (internal)
                               |
                      +--------+---------+
                      |   Go Backend     |
                      |  (Echo Framework)|
                      +--------+---------+
                               |
                        TCP :5432 (internal)
                               |
                      +--------+---------+
                      |   PostgreSQL     |
                      |   Database       |
                      +------------------+
```

### Components

**React Frontend (SPA)**
- Decoupled single-page application built with React.
- Communicates with the backend exclusively via the JSON API.
- Served as static assets through Caddy.
- All client-side routing handled by the SPA; Caddy is configured to return `index.html` for all non-API routes.

**Caddy Reverse Proxy**
- Terminates TLS on port 3443 using either auto-generated self-signed certificates or provisioned certificates from an internal CA.
- Proxies `/api/*` requests to the Go backend on port 8080.
- Serves the React static build for all other paths.
- Handles HTTP security headers: `Strict-Transport-Security`, `X-Content-Type-Options`, `X-Frame-Options`, `Content-Security-Policy`.

**Go Backend (Echo Framework)**
- Stateless HTTP API server built with the Echo web framework for Go.
- Responsible for business logic, authentication, authorization, scheduling conflict detection, and audit log generation.
- Connects to PostgreSQL for all persistent storage.
- Exposes a health check endpoint for container orchestration.

**PostgreSQL Database**
- Single-instance relational database.
- Stores all application data: users, sessions, services, pricing tiers, schedules, staff records, and audit logs.
- Data volume is mounted from the host to ensure persistence across container restarts.

### Inter-Service Communication

All inter-service communication occurs over a Docker internal network. No service port is exposed to the host except Caddy's HTTPS port (3443). The Go backend and PostgreSQL are accessible only within the Docker Compose network.

---

## 3. Security Model

### 3.1 Transport Security

- All client-facing traffic is encrypted via TLS 1.2+ on port 3443, terminated at Caddy.
- Internal traffic between Caddy and the Go backend uses plain HTTP over the Docker internal network, which is considered trusted.
- Caddy sets the `Strict-Transport-Security` header with a minimum `max-age` of one year.

### 3.2 Password Hashing

- All user passwords are hashed using **Argon2id**, the recommended variant of Argon2 that provides resistance against both side-channel and GPU-based attacks.
- Hashing parameters:
  - Memory: 64 MB
  - Iterations: 3
  - Parallelism: 4
  - Salt length: 16 bytes (cryptographically random)
  - Hash length: 32 bytes
- Plaintext passwords are never stored or logged under any circumstance.

### 3.3 Password Policy

- **Minimum length:** 12 characters.
- Passwords are checked against a bundled list of commonly breached passwords at creation and change time.
- Additional complexity requirements (uppercase, digit, special character) are configurable but not enforced by default to encourage passphrase-style passwords.

### 3.4 Authentication Flow

1. The client sends `POST /api/auth/login` with username and password.
2. The server retrieves the user record and verifies the password against the stored Argon2id hash.
3. On success, the server creates a session record in the database and returns a `Set-Cookie` header with a secure, HttpOnly, SameSite=Strict session cookie.
4. Subsequent requests include the session cookie. The server validates the session on every request via middleware.

### 3.5 Brute-Force Protection

- **Failure tracking:** The system tracks consecutive failed login attempts per username, stored in the database with timestamps.
- **CAPTCHA trigger:** After **3 consecutive failed attempts**, the login response includes a CAPTCHA challenge token. The client must present a solved CAPTCHA with the next login attempt.
- **Account lockout:** After **5 failed attempts within a 15-minute window**, the account is locked for 30 minutes. Subsequent login attempts for that account return HTTP 403 regardless of credential validity.
- **Lockout duration:** Accounts are automatically unlocked after **30 minutes**. If the lock has expired, the next login attempt clears the lock state. Administrators may also manually unlock accounts via the user management API.
- **Lockout logging:** Every lockout event is recorded in the audit log with the source IP address.

### 3.6 Session Management

- Sessions are stored server-side in PostgreSQL.
- **Idle timeout:** Sessions expire after **30 minutes** of inactivity. Each authenticated API request resets the idle timer.
- **Absolute timeout:** Sessions have a maximum lifetime of 8 hours regardless of activity.
- **Single session enforcement:** By default, creating a new session invalidates any existing session for the same user.
- The session cookie attributes are:
  - `Secure` -- transmitted only over HTTPS.
  - `HttpOnly` -- not accessible to JavaScript.
  - `SameSite=Strict` -- not sent with cross-origin requests.
  - `Path=/api` -- scoped to API routes only.

### 3.7 HTTP Security Headers

Caddy is configured to add the following headers to all responses:

| Header                    | Value                                              |
|---------------------------|----------------------------------------------------|
| `Strict-Transport-Security` | `max-age=31536000; includeSubDomains`            |
| `X-Content-Type-Options`  | `nosniff`                                          |
| `X-Frame-Options`         | `DENY`                                             |
| `Content-Security-Policy` | `default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'` |
| `Referrer-Policy`         | `strict-origin-when-cross-origin`                  |
| `Permissions-Policy`      | `camera=(), microphone=(), geolocation=()`         |

---

## 4. Identity and Access Management

### 4.1 Role Definitions

The system implements four predefined roles. Roles are not hierarchical; each grants a specific set of permissions.

#### Administrator
- Full access to all system functions.
- Can create, update, and deactivate user accounts.
- Can manage the service catalog and pricing tiers.
- Can manage staff records.
- Can create and modify schedules.
- Can view audit logs.
- At least one Administrator account must exist at all times; the system prevents deactivation of the last Administrator.

#### Scheduler
- Can create, update, and cancel scheduled appointments.
- Can view (but not modify) the service catalog, staff roster, and pricing information.
- Cannot manage users, services, or staff records.
- Cannot view audit logs.

#### Reviewer
- Read-only access to schedules, service catalog, staff roster, and pricing.
- Intended for supervisors or managers who need visibility but should not make changes.
- Cannot manage users or view audit logs.

#### Auditor
- Read-only access to the audit log.
- Read-only access to user accounts (to correlate audit entries with user identities).
- Cannot modify any system data.
- Cannot view schedules, services, or staff details beyond what appears in audit log entries.

### 4.2 Permission Matrix

| Resource            | Administrator | Scheduler | Reviewer | Auditor |
|---------------------|:---:|:---:|:---:|:---:|
| Users (CRUD)        | RW  | --  | --  | R   |
| Service Catalog     | RW  | R   | R   | --  |
| Pricing Tiers       | RW  | R   | R   | --  |
| Staff Roster        | RW  | R   | R   | --  |
| Schedules           | RW  | RW  | R   | --  |
| Audit Logs          | R   | --  | --  | R   |
| System Health       | R   | R   | R   | R   |

*RW = Read/Write, R = Read only, -- = No access*

### 4.3 Role Assignment

- Roles are assigned at account creation time by an Administrator.
- A user holds exactly one role.
- Role changes are recorded in the audit log and take effect on the user's next session (existing sessions are not retroactively modified).

---

## 5. Service Catalog

### 5.1 Service Structure

Each service in the catalog has the following attributes:

- **Name:** Human-readable service name.
- **Category:** Grouping label for organizational purposes (e.g., "inspection", "consultation", "maintenance").
- **Description:** Optional free-text description.
- **Base duration:** The default appointment length in minutes. Must be a multiple of 15.
- **Duration increment:** All appointment durations are specified in 15-minute increments. The frontend enforces this constraint in the scheduling UI.
- **Active flag:** Soft-delete mechanism. Inactive services are hidden from the scheduling interface but preserved in the database for historical reference.

### 5.2 Tiered Pricing

Each service supports multiple pricing tiers based on appointment duration. This allows volume-based or complexity-based pricing models.

**Tier structure:**

| Field                  | Description                                       |
|------------------------|---------------------------------------------------|
| `label`               | Human-readable tier name (e.g., "Standard", "Extended"). |
| `min_duration_minutes` | Minimum appointment duration for this tier.       |
| `max_duration_minutes` | Maximum appointment duration for this tier.       |
| `rate_per_increment`   | Price per 15-minute increment within this tier.   |

Tiers must not have overlapping duration ranges. The system validates tier boundaries on creation and update.

### 5.3 Surcharges

Surcharges are additional fees applied on top of the base tier pricing when specific conditions are met.

**Surcharge types:**

- **Percentage:** A percentage added to the calculated tier price (e.g., 25% after-hours surcharge).
- **Flat:** A fixed amount added to the total (e.g., $50 emergency dispatch fee).

**Common surcharge conditions:**

- After-hours appointments (outside standard 08:00--18:00 window).
- Weekend appointments (Saturday and Sunday).
- Holiday appointments (based on a configurable holiday calendar).
- Emergency or short-notice bookings (less than 24 hours advance notice).

Multiple surcharges can apply to a single appointment. They are calculated independently and summed.

### 5.4 Price Calculation

The total price for an appointment is computed as follows:

```
increments      = duration_minutes / 15
tier            = tier where duration falls within [min_duration, max_duration]
base_price      = increments * tier.rate_per_increment
surcharge_total = sum of applicable surcharges (percentage surcharges applied to base_price)
total_price     = base_price + surcharge_total
```

Price calculations are performed server-side and returned as part of the schedule creation response. The frontend may perform client-side estimates for UI responsiveness, but the server value is authoritative.

---

## 6. Scheduling Engine

### 6.1 Core Concepts

- **Appointment:** A time-bound assignment of a staff member to perform a specific service.
- **Time slot:** Defined by a start time and duration (in 15-minute increments).
- **Staff assignment:** Each appointment is assigned to exactly one staff member.

### 6.2 Conflict Detection

The scheduling engine enforces the following constraints before persisting any new or modified appointment:

#### Rule 1: No Overlapping Appointments

A staff member cannot have two appointments whose time ranges overlap. For a proposed appointment with `start_time` S and `end_time` E (where E = S + duration), there must be no existing appointment for the same staff member where:

```
existing.start_time < E AND existing.end_time > S
```

#### Rule 2: 30-Minute Buffer Between Consecutive Assignments

A mandatory 30-minute buffer is enforced between the end of one appointment and the start of the next for the same staff member. This accounts for travel time, preparation, and administrative overhead.

For conflict checking purposes, the effective occupied time range for an existing appointment is:

```
effective_start = existing.start_time
effective_end   = existing.end_time + 30 minutes
```

A new appointment is valid only if its start time is at or after the `effective_end` of every preceding appointment, and its `effective_end` is at or before the start time of every subsequent appointment.

#### Rule 3: Duration Validity

- Duration must be a positive integer.
- Duration must be a multiple of 15 minutes.
- Duration must fall within the valid range defined by the associated service's pricing tiers.

### 6.3 Conflict Detection Algorithm

When a new appointment is submitted:

1. Query all non-cancelled appointments for the specified `staff_id` that fall within the same calendar day (with a one-day margin on each side to handle edge cases around midnight).
2. For each existing appointment, compute the effective occupied range including the 30-minute buffer.
3. Check if the proposed appointment's time range overlaps with any effective occupied range.
4. If any overlap is detected, return a 409 response with details of all conflicting appointments.
5. If no conflicts exist, persist the appointment and return the created record.

This check is performed within a database transaction with a row-level advisory lock on the staff member's ID to prevent race conditions from concurrent scheduling requests.

### 6.4 Schedule Lifecycle

Appointments follow this state model:

```
  [created] --> confirmed --> cancelled
                    |
                    +--> completed
```

- **Confirmed:** The default state after creation. The appointment is active.
- **Cancelled:** The appointment was cancelled. It no longer blocks the staff member's time and is excluded from future conflict checks.
- **Completed:** The appointment occurred as scheduled. Set manually or via a background job after the appointment's end time has passed.

---

## 7. Audit Logging

### 7.1 Scope

The following events are recorded in the audit log:

- **Authentication:** Successful login, failed login, logout, account lockout.
- **User management:** Create, update, deactivate, role change.
- **Service catalog:** Create, update, deactivate service; create, update, delete pricing tier or surcharge.
- **Staff roster:** Create, update, deactivate staff member.
- **Scheduling:** Create, update, cancel appointment.

### 7.2 Log Entry Structure

Each audit log entry contains:

| Field            | Description                                               |
|------------------|-----------------------------------------------------------|
| `id`             | Unique identifier (UUID v4).                             |
| `timestamp`      | UTC timestamp of the event.                              |
| `actor_id`       | UUID of the user who performed the action.               |
| `actor_username` | Username at the time of the action (denormalized for historical accuracy). |
| `action`         | Dot-notation action identifier (e.g., `user.create`, `schedule.cancel`). |
| `resource_type`  | The type of resource affected (user, service, schedule, etc.). |
| `resource_id`    | UUID of the affected resource.                           |
| `details`        | JSON object containing action-specific data (e.g., changed fields, old and new values). |
| `ip_address`     | Source IP address of the request.                        |

### 7.3 Immutability

- Audit log entries are append-only. The application does not expose any API to modify or delete audit log entries.
- The database table uses `INSERT`-only permissions for the application role; `UPDATE` and `DELETE` are revoked at the PostgreSQL level.
- Each entry includes a SHA-256 hash of the previous entry, forming a hash chain that allows integrity verification.

---

## 8. Deployment Model

### 8.1 Offline-First Architecture

The system is designed to operate entirely without internet connectivity:

- **Container images** are pre-built and distributed as a tarball (`docker save` / `docker load`).
- **Frontend assets** (JavaScript bundles, CSS, fonts, images) are included in the Caddy container as static files. No CDN or external resource references exist.
- **Go binary** is statically compiled with no external runtime dependencies.
- **PostgreSQL** runs from a local container image with data stored on a host-mounted volume.
- **No telemetry or analytics** are included. The system makes zero outbound network requests.

### 8.2 Docker Compose Stack

The deployment consists of four containers managed by Docker Compose:

| Service    | Image Base       | Exposed Port | Purpose                    |
|------------|------------------|:------------:|----------------------------|
| `caddy`    | caddy:2-alpine   | 3443         | TLS termination, static files, reverse proxy |
| `backend`  | golang:1.22-alpine (build stage) | -- (internal 8080/8443) | API server (TLS auto-detected) |
| `frontend` | node (build stage) | -- (internal 80) | React SPA |
| `db`       | postgres:16-alpine | -- (internal 5432) | Database |

Database migrations are applied automatically via Docker entrypoint init scripts mounted from `backend/db/` (01 through 06).

### 8.3 Startup Sequence

1. `db` starts, runs init SQL scripts (`01_init.sql` through `06_enhancements.sql`), and waits for readiness.
2. `backend` starts, connects to PostgreSQL, and begins serving the API on port 8080 (or 8443 with TLS).
3. `frontend` starts, serves the React SPA on port 80.
4. `caddy` starts, loads TLS certificates, and begins proxying traffic on port 3443.

Docker Compose health checks and `depends_on` conditions enforce this ordering.

### 8.4 Initial Setup

On first deployment, the database init scripts create all tables and insert a default Administrator account:

- **Username:** `admin`
- **Password:** `Admin12345!!!` (must be changed on first login in production)

### 8.5 Backup Strategy

- Use `pg_dump` against the running PostgreSQL container to create database backups.
- Backups should be stored on the host filesystem and managed via `cron` or a manual process.
- There is no assumption of internet-based backup services (offline-first design).

---

## 9. Database Schema Overview

### Core Tables

```
users
  id              UUID PRIMARY KEY
  username        VARCHAR(100) UNIQUE NOT NULL
  password_hash   TEXT NOT NULL
  role            VARCHAR(20) NOT NULL
  display_name    VARCHAR(200) NOT NULL
  active          BOOLEAN DEFAULT true
  created_at      TIMESTAMPTZ DEFAULT now()
  updated_at      TIMESTAMPTZ DEFAULT now()

sessions
  id              UUID PRIMARY KEY
  user_id         UUID REFERENCES users(id)
  created_at      TIMESTAMPTZ DEFAULT now()
  last_active_at  TIMESTAMPTZ DEFAULT now()
  expires_at      TIMESTAMPTZ NOT NULL
  ip_address      INET

login_attempts
  id              SERIAL PRIMARY KEY
  username        VARCHAR(100) NOT NULL
  success         BOOLEAN NOT NULL
  ip_address      INET
  attempted_at    TIMESTAMPTZ DEFAULT now()

services
  id                        UUID PRIMARY KEY
  name                      VARCHAR(200) NOT NULL
  category                  VARCHAR(100) NOT NULL
  description               TEXT
  base_duration_minutes     INTEGER NOT NULL
  active                    BOOLEAN DEFAULT true
  created_at                TIMESTAMPTZ DEFAULT now()
  updated_at                TIMESTAMPTZ DEFAULT now()

pricing_tiers
  id                        UUID PRIMARY KEY
  service_id                UUID REFERENCES services(id)
  label                     VARCHAR(100) NOT NULL
  min_duration_minutes      INTEGER NOT NULL
  max_duration_minutes      INTEGER NOT NULL
  rate_per_increment        NUMERIC(10,2) NOT NULL

surcharges
  id              UUID PRIMARY KEY
  service_id      UUID REFERENCES services(id)
  name            VARCHAR(100) NOT NULL
  type            VARCHAR(20) NOT NULL  -- 'percentage' or 'flat'
  value           NUMERIC(10,2) NOT NULL
  conditions      TEXT

staff
  id              UUID PRIMARY KEY
  employee_id     VARCHAR(50) UNIQUE NOT NULL
  display_name    VARCHAR(200) NOT NULL
  department      VARCHAR(100) NOT NULL
  qualifications  JSONB DEFAULT '[]'
  active          BOOLEAN DEFAULT true
  created_at      TIMESTAMPTZ DEFAULT now()
  updated_at      TIMESTAMPTZ DEFAULT now()

schedules
  id                UUID PRIMARY KEY
  service_id        UUID REFERENCES services(id)
  staff_id          UUID REFERENCES staff(id)
  start_time        TIMESTAMPTZ NOT NULL
  end_time          TIMESTAMPTZ NOT NULL
  duration_minutes  INTEGER NOT NULL
  status            VARCHAR(20) DEFAULT 'confirmed'
  notes             TEXT
  created_by        UUID REFERENCES users(id)
  created_at        TIMESTAMPTZ DEFAULT now()
  updated_at        TIMESTAMPTZ DEFAULT now()

audit_logs
  id                UUID PRIMARY KEY
  timestamp         TIMESTAMPTZ DEFAULT now()
  actor_id          UUID
  actor_username    VARCHAR(100)
  action            VARCHAR(100) NOT NULL
  resource_type     VARCHAR(50) NOT NULL
  resource_id       UUID
  details           JSONB
  ip_address        INET
  prev_hash         CHAR(64)
```

### Key Indexes

- `users(username)` -- unique, used for login lookups.
- `sessions(user_id)` -- used for session invalidation on new login.
- `login_attempts(username, attempted_at)` -- used for brute-force detection queries.
- `schedules(staff_id, start_time, end_time)` -- used for conflict detection queries.
- `schedules(status)` -- used for filtering active appointments.
- `audit_logs(timestamp)` -- used for chronological retrieval.
- `audit_logs(actor_id)` -- used for per-user audit queries.
- `audit_logs(resource_type, resource_id)` -- used for per-resource audit queries.
