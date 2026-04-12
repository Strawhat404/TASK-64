# Local Operations & Compliance Console -- API Specification

**Version:** 2.1.0
**Base URL:** `https://<host>:8443/api`
**Content-Type:** All requests and responses use `application/json` unless otherwise noted (e.g. file uploads use `multipart/form-data`, exports return binary files).
**Authentication:** Session-based via `Set-Cookie` / `Cookie` headers. All endpoints except `POST /api/auth/login` and `GET /api/health` require a valid session cookie.
**Authorization:** Role-based access control (RBAC). Each endpoint documents its required role(s). Roles are: **Administrator**, **Scheduler**, **Reviewer**, **Auditor**.
**Multi-Tenancy:** All data is scoped to a tenant. The tenant is derived from the authenticated user's session.

---

## 1. Authentication

### 1.1 POST /api/auth/login

Authenticate a user and establish a session.

**Access:** Public (no session required). Rate-limited to 10 attempts per 15 minutes. CAPTCHA required after 3 consecutive failures.

**Request Body:**

| Field      | Type   | Required | Description          |
|------------|--------|----------|----------------------|
| `username` | string | yes      | Account username.    |
| `password` | string | yes      | Account password (minimum 12 characters). |

**Success Response (200):**

```json
{
  "user": {
    "id": "uuid",
    "tenant_id": "uuid",
    "role_id": 1,
    "role_name": "Administrator",
    "username": "jdoe",
    "email": "jdoe@example.com",
    "failed_login_attempts": 0,
    "captcha_required": false,
    "last_login_at": "2026-04-09T12:00:00Z",
    "created_at": "2026-01-15T10:00:00Z",
    "updated_at": "2026-03-20T14:22:00Z",
    "is_active": true
  }
}
```

A `Set-Cookie` header with a secure, HttpOnly, SameSite=Strict session cookie is included.

**Error Responses:**

| Status | Condition                                           |
|--------|-----------------------------------------------------|
| 400    | Password shorter than 12 characters.                |
| 401    | Invalid credentials.                                |
| 403    | Account locked or deactivated.                      |

---

### 1.2 POST /api/auth/logout

Destroy the current session.

**Access:** Any authenticated user.

**Request Body:** None.

**Success Response (200):**

```json
{
  "message": "Logged out"
}
```

The session cookie is cleared via `Set-Cookie` with an expired date.

---

### 1.3 GET /api/auth/session

Return the currently authenticated user's session information. Used by the frontend to validate an existing cookie on page load.

**Access:** Any authenticated user.

**Success Response (200):**

```json
{
  "user": {
    "id": "uuid",
    "tenant_id": "uuid",
    "role_id": 1,
    "role_name": "Administrator",
    "username": "jdoe",
    "email": "jdoe@example.com",
    "is_active": true,
    "created_at": "2026-01-15T10:00:00Z",
    "updated_at": "2026-03-20T14:22:00Z"
  }
}
```

**Error Responses:**

| Status | Condition                        |
|--------|----------------------------------|
| 401    | No valid session / session expired. |

---

## 2. User Management

All user management endpoints require the **Administrator** role.

### 2.1 GET /api/users

List all user accounts.

**Query Parameters:**

| Param    | Type    | Default | Description                       |
|----------|---------|---------|-----------------------------------|
| `page`   | integer | 1       | Page number.                      |
| `limit`  | integer | 25      | Items per page (max 100).         |
| `role`   | string  | --      | Filter by role.                   |
| `search` | string  | --      | Search by username or email.      |

**Success Response (200):**

```json
[
  {
    "id": "uuid",
    "tenant_id": "uuid",
    "role_id": 1,
    "role_name": "Administrator",
    "username": "jdoe",
    "email": "jdoe@example.com",
    "failed_login_attempts": 0,
    "captcha_required": false,
    "last_login_at": "2026-04-09T12:00:00Z",
    "is_active": true,
    "created_at": "2026-01-15T10:00:00Z",
    "updated_at": "2026-03-20T14:22:00Z"
  }
]
```

---

### 2.2 POST /api/users

Create a new user account.

**Request Body:**

| Field      | Type    | Required | Description                                  |
|------------|---------|----------|----------------------------------------------|
| `username` | string  | yes      | Unique username.                             |
| `email`    | string  | yes      | Unique email address.                        |
| `password` | string  | yes      | Minimum 12 characters.                       |
| `role_id`  | integer | yes      | Role ID (foreign key to roles table).        |

**Success Response (201):** Created user object (same shape as GET response).

**Error Responses:**

| Status | Condition                              |
|--------|----------------------------------------|
| 400    | Validation failure (weak password, missing field). |
| 409    | Username or email already exists.      |

---

### 2.3 GET /api/users/:id

Retrieve a single user by ID.

**Success Response (200):** Single user object.

**Error Responses:**

| Status | Condition       |
|--------|-----------------|
| 404    | User not found. |

---

### 2.4 PUT /api/users/:id

Update an existing user account. Partial updates are accepted.

**Request Body:** Any combination of `username`, `email`, `role_id`, `is_active`.

**Success Response (200):** Updated user object.

**Error Responses:**

| Status | Condition                                |
|--------|------------------------------------------|
| 400    | Validation failure.                      |
| 404    | User not found.                          |
| 409    | Username or email conflict.              |

---

### 2.5 DELETE /api/users/:id

Deactivate a user account. Accounts are soft-deleted (set to `is_active: false`) to preserve audit trail integrity.

**Success Response (200):**

```json
{
  "message": "User deactivated"
}
```

**Error Responses:**

| Status | Condition       |
|--------|-----------------|
| 404    | User not found. |

---

## 3. Service Catalog

**Access:** Read operations available to all authenticated users. Write operations require **Administrator** or **Scheduler** role. Delete requires **Administrator**.

### 3.1 GET /api/services

List all services in the catalog.

**Query Parameters:**

| Param      | Type    | Default | Description                        |
|------------|---------|---------|------------------------------------|
| `page`     | integer | 1       | Page number.                       |
| `limit`    | integer | 25      | Items per page (max 100).          |
| `category` | string  | --      | Filter by service category.        |
| `active`   | boolean | --      | Filter by active status.           |

**Success Response (200):**

```json
[
  {
    "id": "uuid",
    "tenant_id": "uuid",
    "name": "Standard Inspection",
    "description": "Routine compliance inspection.",
    "base_price_usd": 150.00,
    "tier": "standard",
    "after_hours_surcharge_pct": 25,
    "same_day_surcharge_usd": 50.00,
    "duration_minutes": 60,
    "is_active": true,
    "headcount": 2,
    "required_tools": ["inspection-kit", "camera"],
    "add_ons": null,
    "daily_cap": 10,
    "created_at": "2026-01-10T08:00:00Z",
    "updated_at": "2026-01-10T08:00:00Z"
  }
]
```

---

### 3.2 POST /api/services

Create a new service entry. **Requires Administrator or Scheduler role.**

**Request Body:**

| Field                       | Type     | Required | Description                             |
|-----------------------------|----------|----------|-----------------------------------------|
| `name`                      | string   | yes      | Service name.                           |
| `description`               | string   | no       | Free-text description.                  |
| `base_price_usd`            | number   | yes      | Base price in USD.                      |
| `tier`                      | string   | yes      | One of: standard, premium, enterprise.  |
| `after_hours_surcharge_pct` | integer  | no       | After-hours surcharge percentage.       |
| `same_day_surcharge_usd`    | number   | no       | Same-day booking surcharge in USD.      |
| `duration_minutes`          | integer  | yes      | Duration in minutes.                    |
| `headcount`                 | integer  | no       | Staff required. Defaults to 1.          |
| `required_tools`            | string[] | no       | List of required tool names.            |
| `add_ons`                   | object   | no       | Optional add-on configuration.          |
| `daily_cap`                 | integer  | no       | Maximum daily bookings (null = no limit).|

**Success Response (201):** Created service object.

---

### 3.3 GET /api/services/:id

Retrieve a single service by ID.

**Success Response (200):** Single service object.

**Error Responses:**

| Status | Condition          |
|--------|--------------------|
| 404    | Service not found. |

---

### 3.4 GET /api/services/:id/pricing

Retrieve the calculated pricing for a specific service, including tier multiplier and surcharges.

**Access:** All authenticated users.

**Success Response (200):**

```json
{
  "service_id": "uuid",
  "service_name": "Standard Inspection",
  "base_price_usd": 150.00,
  "tier_multiplier": 1.0,
  "tier_adjusted_price": 150.00,
  "after_hours_surcharge": 37.50,
  "same_day_surcharge": 50.00,
  "total_usd": 237.50
}
```

---

### 3.5 PUT /api/services/:id

Update a service. **Requires Administrator or Scheduler role.**

**Request Body:** Any mutable service fields (name, description, base_price_usd, tier, after_hours_surcharge_pct, same_day_surcharge_usd, duration_minutes, is_active, headcount, required_tools, add_ons, daily_cap).

**Success Response (200):** Updated service object.

---

### 3.6 DELETE /api/services/:id

Soft-delete a service (sets `is_active: false`). **Requires Administrator role.**

**Success Response (200):**

```json
{
  "message": "Service deactivated"
}
```

---

## 4. Scheduling

**Access:** All authenticated users for read. **Scheduler** and **Administrator** roles for write operations.

### 4.1 GET /api/schedules

List scheduled appointments with filtering.

**Query Parameters:**

| Param        | Type    | Default | Description                              |
|--------------|---------|---------|------------------------------------------|
| `page`       | integer | 1       | Page number.                             |
| `limit`      | integer | 25      | Items per page (max 100).                |
| `staff_id`   | string  | --      | Filter by staff member.                  |
| `start_date` | string  | --      | ISO 8601 date, inclusive lower bound.    |
| `end_date`   | string  | --      | ISO 8601 date, inclusive upper bound.    |
| `status`     | string  | --      | Filter by status (pending, confirmed, in_progress, completed, cancelled). |

**Success Response (200):** Array of schedule objects.

```json
[
  {
    "id": "uuid",
    "tenant_id": "uuid",
    "service_id": "uuid",
    "staff_id": "uuid",
    "client_name": "Acme Corp",
    "scheduled_start": "2026-04-10T09:00:00Z",
    "scheduled_end": "2026-04-10T10:00:00Z",
    "status": "confirmed",
    "requires_confirmation": true,
    "confirmed_at": "2026-04-09T15:00:00Z",
    "reassignment_reason": null,
    "reassignment_reason_code": null,
    "created_at": "2026-04-09T12:00:00Z",
    "updated_at": "2026-04-09T12:00:00Z"
  }
]
```

---

### 4.2 POST /api/schedules

Create a new scheduled appointment. The server performs conflict detection including a mandatory 30-minute buffer between consecutive assignments. **Requires Administrator or Scheduler role.**

**Request Body:**

| Field                  | Type    | Required | Description                                     |
|------------------------|---------|----------|-------------------------------------------------|
| `service_id`           | string  | yes      | UUID of the service.                            |
| `staff_id`             | string  | yes      | UUID of the assigned staff member.              |
| `client_name`          | string  | yes      | Name of the client.                             |
| `scheduled_start`      | string  | yes      | ISO 8601 datetime.                              |
| `scheduled_end`        | string  | yes      | ISO 8601 datetime. Must be after scheduled_start.|
| `requires_confirmation`| boolean | no       | Whether staff confirmation is required.          |

**Conflict Checking Rules:**
- No overlapping appointments for the same staff member.
- A mandatory 30-minute buffer is enforced between consecutive assignments for the same staff member.
- If a conflict is detected the request is rejected with details.

**Success Response (201):** Created schedule object.

**Error Responses:**

| Status | Condition                                      |
|--------|------------------------------------------------|
| 400    | Validation failure (missing fields, invalid times). |
| 409    | Scheduling conflict detected.                  |

---

### 4.3 PUT /api/schedules/:id

Update a schedule entry. Re-runs conflict detection if time or staff assignment changes. **Requires Administrator or Scheduler role.**

**Request Body:** Any combination of `service_id`, `staff_id`, `client_name`, `scheduled_start`, `scheduled_end`, `status`, `reassignment_reason`, `reassignment_reason_code`.

**Success Response (200):** Updated schedule object.

---

### 4.4 DELETE /api/schedules/:id

Cancel a scheduled appointment (sets `status: "cancelled"`). **Requires Administrator or Scheduler role.**

**Success Response (200):**

```json
{
  "message": "Schedule cancelled"
}
```

---

### 4.5 POST /api/schedules/:id/confirm

Confirm a staff assignment for a schedule that requires confirmation.

**Access:** Any authenticated user (typically the assigned staff member).

**Success Response (200):** Updated schedule object with `confirmed_at` set.

---

### 4.6 GET /api/schedules/available-staff

Find staff members who are available (not booked) during a given time range, with optional specialization and location filtering.

**Access:** All authenticated users.

**Query Parameters:**

| Param            | Type   | Required | Description                             |
|------------------|--------|----------|-----------------------------------------|
| `start`          | string | yes      | ISO 8601 datetime, start of range.      |
| `end`            | string | yes      | ISO 8601 datetime, end of range.        |
| `specialization` | string | no       | Filter by staff specialization.         |
| `location_id`    | string | no       | Filter by location (ABAC).             |

**Success Response (200):** Array of available staff member objects.

---

### 4.7 POST /api/schedules/:id/backup

Request a backup/substitute staff member for an existing schedule. **Requires Administrator or Scheduler role.**

**Request Body:**

| Field            | Type   | Required | Description                          |
|------------------|--------|----------|--------------------------------------|
| `backup_staff_id`| string | yes      | UUID of the backup staff member.     |
| `reason_code`    | string | yes      | Reason code for the backup request.  |
| `notes`          | string | no       | Additional notes.                    |

**Success Response (201):** Created backup staff assignment object.

```json
{
  "id": "uuid",
  "schedule_id": "uuid",
  "primary_staff_id": "uuid",
  "backup_staff_id": "uuid",
  "reason_code": "illness",
  "notes": "Staff member called in sick",
  "status": "pending",
  "confirmed_at": null,
  "created_by": "uuid",
  "created_at": "2026-04-09T12:00:00Z"
}
```

---

### 4.8 POST /api/schedules/backup/:id/confirm

Confirm a pending backup staff assignment. **Requires Administrator or Scheduler role.**

**Success Response (200):** Updated backup assignment object with `status: "confirmed"` and `confirmed_at` set.

---

## 5. Staff Roster

**Access:** Read operations available to all authenticated users. Write operations require **Administrator** or **Scheduler** role. Delete requires **Administrator**.

### 5.1 GET /api/staff

List staff members.

**Query Parameters:**

| Param    | Type    | Default | Description                      |
|----------|---------|---------|----------------------------------|
| `page`   | integer | 1       | Page number.                     |
| `limit`  | integer | 25      | Items per page (max 100).        |

**Success Response (200):**

```json
[
  {
    "id": "uuid",
    "tenant_id": "uuid",
    "user_id": "uuid",
    "full_name": "Alex Rivera",
    "specialization": "inspection-level-2",
    "is_available": true,
    "created_at": "2025-11-01T08:00:00Z",
    "updated_at": "2025-11-01T08:00:00Z"
  }
]
```

---

### 5.2 POST /api/staff

Create a new staff record. **Requires Administrator or Scheduler role.**

**Request Body:**

| Field            | Type   | Required | Description                    |
|------------------|--------|----------|--------------------------------|
| `user_id`        | string | yes      | UUID of the associated user.   |
| `full_name`      | string | yes      | Full name.                     |
| `specialization` | string | yes      | Specialization/qualification.  |

**Success Response (201):** Created staff object.

---

### 5.3 GET /api/staff/:id

Retrieve a single staff member.

**Success Response (200):** Single staff object.

---

### 5.4 PUT /api/staff/:id

Update staff details. **Requires Administrator or Scheduler role.**

**Request Body:** Any combination of `full_name`, `specialization`, `is_available`.

**Success Response (200):** Updated staff object.

---

### 5.5 DELETE /api/staff/:id

Deactivate a staff member (soft-delete). **Requires Administrator role.**

**Success Response (200):**

```json
{
  "message": "Staff member deactivated"
}
```

---

### 5.6 GET /api/staff/:id/credentials

List credentials/certifications for a staff member.

**Access:** All authenticated users.

**Success Response (200):**

```json
[
  {
    "id": "uuid",
    "staff_id": "uuid",
    "credential_name": "OSHA Safety Certification",
    "issuing_authority": "OSHA",
    "credential_number": "OSH-12345",
    "issued_date": "2025-01-15T00:00:00Z",
    "expiry_date": "2027-01-15T00:00:00Z",
    "status": "active",
    "created_at": "2025-01-15T10:00:00Z",
    "updated_at": "2025-01-15T10:00:00Z"
  }
]
```

---

### 5.7 POST /api/staff/:id/credentials

Add a credential to a staff member. **Requires Administrator or Scheduler role.**

**Request Body:**

| Field               | Type   | Required | Description                       |
|---------------------|--------|----------|-----------------------------------|
| `credential_name`   | string | yes      | Name of the credential.           |
| `issuing_authority` | string | no       | Authority that issued it.         |
| `credential_number` | string | no       | Unique credential number.         |
| `issued_date`       | string | no       | ISO 8601 date.                    |
| `expiry_date`       | string | no       | ISO 8601 date.                    |
| `status`            | string | no       | Defaults to "active".             |

**Success Response (201):** Created credential object.

---

### 5.8 GET /api/staff/:id/availability

List availability windows for a staff member.

**Access:** All authenticated users.

**Success Response (200):**

```json
[
  {
    "id": "uuid",
    "staff_id": "uuid",
    "day_of_week": 1,
    "start_time": "08:00",
    "end_time": "17:00",
    "is_recurring": true,
    "specific_date": null,
    "created_at": "2025-11-01T08:00:00Z"
  }
]
```

---

### 5.9 POST /api/staff/:id/availability

Add an availability window for a staff member. **Requires Administrator or Scheduler role.**

**Request Body:**

| Field           | Type    | Required | Description                                |
|-----------------|---------|----------|--------------------------------------------|
| `day_of_week`   | integer | yes      | 0 (Sunday) through 6 (Saturday).          |
| `start_time`    | string  | yes      | HH:MM format.                             |
| `end_time`      | string  | yes      | HH:MM format.                             |
| `is_recurring`  | boolean | no       | Whether this is a recurring window.        |
| `specific_date` | string  | no       | ISO 8601 date for non-recurring windows.   |

**Success Response (201):** Created availability object.

---

## 6. Audit Logs

### 6.1 GET /api/audit/logs

Retrieve the system audit log. **Requires Auditor or Administrator role.**

Every state-changing operation (create, update, delete, login, logout, failed login) is recorded.

**Query Parameters:**

| Param       | Type    | Default | Description                                  |
|-------------|---------|---------|----------------------------------------------|
| `page`      | integer | 1       | Page number.                                 |
| `limit`     | integer | 50      | Items per page (max 200).                    |
| `actor_id`  | string  | --      | Filter by user who performed the action.     |
| `action`    | string  | --      | Filter by action type (e.g. create_user).    |
| `resource`  | string  | --      | Filter by resource type (user, service, etc).|
| `date_from` | string  | --      | ISO 8601 date, inclusive lower bound.        |
| `date_to`   | string  | --      | ISO 8601 date, inclusive upper bound.        |

**Success Response (200):**

```json
[
  {
    "id": "uuid",
    "tenant_id": "uuid",
    "user_id": "uuid",
    "action": "create_schedule",
    "resource_type": "schedule",
    "resource_id": "uuid",
    "details": "{\"client_name\":\"Acme Corp\"}",
    "ip_address": "10.0.1.50",
    "created_at": "2026-04-09T11:42:00Z"
  }
]
```

---

## 7. Health Check

### 7.1 GET /api/health

Returns system health status. Used by monitoring and Docker health checks.

**Access:** Public (no authentication required).

**Success Response (200):**

```json
{
  "status": "ok",
  "version": "1.0.0",
  "uptime_seconds": 86400,
  "database": "connected",
  "timestamp": "2026-04-09T12:00:00Z"
}
```

**Degraded Response (503):**

```json
{
  "status": "degraded",
  "version": "1.0.0",
  "uptime_seconds": 86400,
  "database": "disconnected",
  "timestamp": "2026-04-09T12:00:00Z"
}
```

---

## 8. Content Governance

Content governance endpoints manage content lifecycle, multi-level review pipelines, gray-release rollout, automated rule enforcement, version history, resource relationships, and re-review workflows.

### 8.1 POST /api/governance/content

Create new governed content. **Requires Administrator or Reviewer role.**

**Request Body:**

| Field         | Type   | Required | Description                   |
|---------------|--------|----------|-------------------------------|
| `title`       | string | yes      | Content title.                |
| `body`        | string | yes      | Content body.                 |
| `content_type`| string | no       | Type classification.          |

**Success Response (201):** Created content object.

---

### 8.2 GET /api/governance/content

List all governed content with pagination.

**Access:** All authenticated users.

**Query Parameters:**

| Param    | Type    | Default | Description           |
|----------|---------|---------|-----------------------|
| `page`   | integer | 1       | Page number.          |
| `limit`  | integer | 25      | Items per page.       |
| `status` | string  | --      | Filter by status.     |

**Success Response (200):** Paginated array of content objects.

---

### 8.3 GET /api/governance/content/:id

Retrieve a single content item by ID.

**Access:** All authenticated users.

**Success Response (200):** Single content object.

---

### 8.4 PUT /api/governance/content/:id

Update a content item. Creates a new version. **Requires Administrator or Reviewer role.**

**Success Response (200):** Updated content object.

---

### 8.5 POST /api/governance/content/:id/submit

Submit content for multi-level review. **Requires Administrator or Reviewer role.**

**Success Response (200):** Content object with updated review status.

---

### 8.6 POST /api/governance/content/:id/promote

Promote gray-release content to full production. **Requires Administrator role.**

**Success Response (200):** Promoted content object.

---

### 8.7 GET /api/governance/content/:id/versions

Get version history for a content item.

**Access:** All authenticated users.

**Success Response (200):** Array of version objects.

---

### 8.8 POST /api/governance/content/:id/rollback

Rollback content to a previous version. **Requires Administrator role.**

**Success Response (200):** Rolled-back content object.

---

### 8.9 GET /api/governance/content/:id/versions/diff

Diff two versions of a content item. **Requires Administrator or Reviewer role.**

**Query Parameters:**

| Param      | Type    | Required | Description                 |
|------------|---------|----------|-----------------------------|
| `version1` | integer | yes      | First version number.       |
| `version2` | integer | yes      | Second version number.      |

**Success Response (200):** Diff output comparing the two versions.

---

### 8.10 POST /api/governance/content/:id/re-review

Trigger a re-review of previously approved content. **Requires Administrator or Reviewer role.**

**Success Response (200):** Content object with review status reset.

---

### 8.11 GET /api/governance/reviews/pending

List content items pending review. **Requires Administrator or Reviewer role.**

**Success Response (200):** Array of content objects awaiting review.

---

### 8.12 POST /api/governance/reviews/:id/decide

Submit a review decision. **Requires Administrator or Reviewer role.**

**Request Body:**

| Field            | Type    | Required | Description                          |
|------------------|---------|----------|--------------------------------------|
| `decision`       | string  | yes      | One of: approve, reject, escalate.   |
| `decision_notes` | string  | no       | Reviewer notes.                      |
| `level`          | integer | yes      | Current review level being completed.|

**Success Response (200):** Updated review status.

---

### 8.13 GET /api/governance/gray-release

List content items currently in gray-release.

**Access:** All authenticated users.

**Success Response (200):** Array of gray-release content objects.

---

### 8.14 GET /api/governance/rules

List all governance rules. **Requires Administrator role.**

**Success Response (200):** Array of rule objects.

---

### 8.15 POST /api/governance/rules

Create a governance rule. **Requires Administrator role.**

**Request Body:**

| Field       | Type   | Required | Description                          |
|-------------|--------|----------|--------------------------------------|
| `name`      | string | yes      | Rule name.                           |
| `pattern`   | string | yes      | Regex or keyword pattern.            |
| `action`    | string | yes      | Action on match (block, flag, etc.). |
| `severity`  | string | no       | Severity level.                      |

**Success Response (201):** Created rule object.

---

### 8.16 PUT /api/governance/rules/:id

Update a governance rule. **Requires Administrator role.**

**Success Response (200):** Updated rule object.

---

### 8.17 DELETE /api/governance/rules/:id

Delete a governance rule. **Requires Administrator role.**

**Success Response (200):**

```json
{
  "message": "Rule deleted"
}
```

---

### 8.18 POST /api/governance/relationships

Create a relationship between content resources. **Requires Administrator or Reviewer role.**

**Request Body:**

| Field              | Type   | Required | Description                           |
|--------------------|--------|----------|---------------------------------------|
| `source_id`        | string | yes      | UUID of the source content.           |
| `target_id`        | string | yes      | UUID of the target content.           |
| `relationship_type`| string | yes      | Type of relationship (depends_on, related_to, etc.). |

**Success Response (201):** Created relationship object.

---

### 8.19 GET /api/governance/relationships

List content relationships.

**Access:** All authenticated users.

**Query Parameters:**

| Param        | Type   | Default | Description                    |
|--------------|--------|---------|--------------------------------|
| `content_id` | string | --      | Filter by content ID.          |

**Success Response (200):** Array of relationship objects.

---

### 8.20 DELETE /api/governance/relationships/:id

Delete a content relationship. **Requires Administrator role.**

**Success Response (200):**

```json
{
  "message": "Relationship deleted"
}
```

---

## 9. Financial Reconciliation

**Access:** All authenticated users for read. **Administrator** and **Auditor** roles for import and matching.

Financial reconciliation endpoints handle transaction feed import, automated matching with a $1.00 variance threshold, duplicate detection, and exception management.

### 9.1 POST /api/reconciliation/import

Import a transaction feed from CSV file. **Requires Administrator or Auditor role.**

**Content-Type:** `multipart/form-data`

**Form Fields:**

| Field       | Type   | Required | Description                              |
|-------------|--------|----------|------------------------------------------|
| `file`      | file   | yes      | CSV file with transaction data.          |
| `feed_type` | string | yes      | One of: internal, external.              |

**Success Response (200):**

```json
{
  "feed_id": "uuid",
  "records_imported": 150,
  "records_matched": 0,
  "exceptions_generated": 0
}
```

**Error Responses:**

| Status | Condition                                |
|--------|------------------------------------------|
| 400    | Invalid CSV format or missing columns.   |
| 413    | File exceeds maximum upload size.        |

---

### 9.2 GET /api/reconciliation/feeds

List imported reconciliation feeds.

**Access:** All authenticated users.

**Success Response (200):** Array of feed summary objects.

```json
[
  {
    "id": "uuid",
    "tenant_id": "uuid",
    "filename": "transactions_april.csv",
    "feed_type": "internal",
    "record_count": 150,
    "imported_by": "uuid",
    "imported_at": "2026-04-09T12:00:00Z",
    "status": "completed"
  }
]
```

---

### 9.3 GET /api/reconciliation/feeds/:id

Retrieve a single feed by ID.

**Success Response (200):** Single feed object.

---

### 9.4 POST /api/reconciliation/feeds/:id/match

Trigger automated matching for a reconciliation feed. **Requires Administrator or Auditor role.**

**Success Response (200):**

```json
{
  "total_internal": 150,
  "total_external": 145,
  "matched_count": 120,
  "exception_count": 25,
  "matches": [],
  "exceptions": []
}
```

---

### 9.5 GET /api/reconciliation/matches

List transaction match results.

**Access:** All authenticated users.

**Success Response (200):** Array of match objects.

---

### 9.6 GET /api/reconciliation/exceptions

List reconciliation exceptions requiring attention.

**Access:** All authenticated users.

**Query Parameters:**

| Param            | Type    | Default | Description                                              |
|------------------|---------|---------|----------------------------------------------------------|
| `page`           | integer | 1       | Page number.                                             |
| `limit`          | integer | 25      | Items per page.                                          |
| `exception_type` | string  | --      | Filter: variance_over_threshold, duplicate_suspect, unmatched. |
| `status`         | string  | --      | Filter: open, in_progress, resolved.                     |

**Success Response (200):**

```json
{
  "data": [
    {
      "id": "uuid",
      "tenant_id": "uuid",
      "transaction_id": "uuid",
      "match_id": "uuid",
      "exception_type": "variance_over_threshold",
      "severity": "medium",
      "amount": 2200.00,
      "variance_amount": 1.50,
      "description": "Amount variance exceeds $1.00 threshold",
      "assigned_to": null,
      "status": "open",
      "disposition": null,
      "resolution_notes": null,
      "resolved_by": null,
      "resolved_at": null,
      "created_at": "2026-04-09T12:00:00Z",
      "updated_at": "2026-04-09T12:00:00Z"
    }
  ],
  "page": 1,
  "per_page": 25,
  "total": 8,
  "total_pages": 1
}
```

---

### 9.7 GET /api/reconciliation/exceptions/:id

Retrieve a single exception by ID.

**Success Response (200):** Single exception object.

---

### 9.8 GET /api/reconciliation/exceptions/export

Export reconciliation exceptions in CSV or XLSX format.

**Access:** All authenticated users.

**Query Parameters:**

| Param    | Type   | Default | Description                                 |
|----------|--------|---------|---------------------------------------------|
| `format` | string | csv     | Export format: `csv` or `xlsx`.             |

**Success Response (200):** Binary file download with appropriate `Content-Type` header (`text/csv` or `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`).

---

### 9.9 PUT /api/reconciliation/exceptions/:id/assign

Assign an exception to a user for resolution. **Requires Administrator or Auditor role.**

**Request Body:**

| Field         | Type   | Required | Description                    |
|---------------|--------|----------|--------------------------------|
| `assigned_to` | string | yes      | UUID of the user to assign to. |

**Success Response (200):** Updated exception object.

---

### 9.10 PUT /api/reconciliation/exceptions/:id/resolve

Resolve a reconciliation exception. **Requires Administrator or Auditor role.**

**Request Body:**

| Field              | Type   | Required | Description                             |
|--------------------|--------|----------|-----------------------------------------|
| `disposition`      | string | yes      | One of: approved, rejected, escalated.  |
| `resolution_notes` | string | no       | Resolution notes.                       |

**Success Response (200):** Updated exception object with resolution details.

---

### 9.11 GET /api/reconciliation/summary

Get overall reconciliation statistics for the current tenant.

**Access:** All authenticated users.

**Success Response (200):**

```json
{
  "total_feeds": 5,
  "total_transactions": 750,
  "total_matched": 680,
  "total_exceptions": 45,
  "open_exceptions": 12,
  "total_open": 12,
  "unmatched_items": 8,
  "suspected_duplicates": 3,
  "variance_alerts": 5,
  "match_rate_pct": 90.67,
  "match_rate": 90.67
}
```

---

## 10. Security and Encryption

All security endpoints require the **Administrator** role.

### 10.1 POST /api/security/sensitive

Store a sensitive data field using AES-256-GCM encryption.

**Request Body:**

| Field       | Type   | Required | Description                    |
|-------------|--------|----------|--------------------------------|
| `field_name`| string | yes      | Field name to store.           |
| `value`     | string | yes      | Plaintext value to encrypt.    |

**Success Response (201):** Created sensitive data record (encrypted, plaintext not returned).

---

### 10.2 GET /api/security/sensitive

List stored sensitive data fields (metadata only, values are masked).

**Success Response (200):** Array of sensitive data metadata objects.

---

### 10.3 GET /api/security/sensitive/:id

Retrieve metadata for a specific sensitive data record (value is masked).

**Success Response (200):** Sensitive data metadata object.

---

### 10.4 POST /api/security/sensitive/:id/reveal

Decrypt and reveal a sensitive data field value. Creates an audit log entry.

**Success Response (200):** Object with decrypted value.

---

### 10.5 DELETE /api/security/sensitive/:id

Delete a sensitive data record.

**Success Response (200):**

```json
{
  "message": "Sensitive data deleted"
}
```

---

### 10.6 POST /api/security/keys/rotate

Trigger encryption key rotation. All encrypted fields are re-encrypted with the new key.

**Request Body:**

| Field    | Type   | Required | Description                          |
|----------|--------|----------|--------------------------------------|
| `reason` | string | yes      | Reason for rotation (audit trail).   |

**Success Response (200):** Key rotation result with previous and new key versions.

---

### 10.7 GET /api/security/keys

Retrieve current encryption key metadata.

**Success Response (200):**

```json
{
  "current_key_version": 4,
  "created_at": "2026-04-09T12:05:00Z",
  "algorithm": "AES-256-GCM",
  "total_encrypted_fields": 1247,
  "last_rotation": "2026-04-09T12:05:00Z"
}
```

---

### 10.8 GET /api/security/keys/rotation-due

Check if key rotation is overdue based on policy.

**Success Response (200):** Object indicating whether rotation is due and when the last rotation occurred.

---

### 10.9 GET /api/security/audit-ledger

Retrieve the cryptographic audit ledger (tamper-evident log).

**Success Response (200):** Array of ledger entries with hash chain.

---

### 10.10 POST /api/security/audit-ledger/verify

Verify the integrity of the audit ledger hash chain.

**Success Response (200):** Verification result indicating whether the chain is intact.

---

### 10.11 GET /api/security/retention

Get data retention policies.

**Success Response (200):** Array of retention policy objects.

---

### 10.12 POST /api/security/retention/cleanup

Run data retention cleanup according to configured policies.

**Success Response (200):** Cleanup result with counts of records purged.

---

### 10.13 GET /api/security/rate-limits

Get current rate limit status.

**Success Response (200):** Rate limit status information.

---

### 10.14 POST /api/security/legal-holds

Create a legal hold to prevent data deletion.

**Request Body:**

| Field       | Type   | Required | Description                    |
|-------------|--------|----------|--------------------------------|
| `reason`    | string | yes      | Reason for the hold.           |
| `scope`     | string | yes      | Scope of the hold.             |

**Success Response (201):** Created legal hold object.

---

### 10.15 GET /api/security/legal-holds

List all legal holds.

**Success Response (200):** Array of legal hold objects.

---

### 10.16 PUT /api/security/legal-holds/:id/release

Release (lift) a legal hold.

**Success Response (200):** Updated legal hold object with released status.

---

## Common Error Format

All error responses follow a consistent structure:

```json
{
  "error": "Human-readable description of the problem."
}
```

## HTTP Status Code Summary

| Code | Usage                                             |
|------|---------------------------------------------------|
| 200  | Successful read or update.                        |
| 201  | Successful resource creation.                     |
| 400  | Validation error or malformed request.            |
| 401  | Missing or expired session.                       |
| 403  | Authenticated but insufficient role permissions or account locked. |
| 404  | Resource not found.                               |
| 409  | Conflict (duplicate username, scheduling clash).  |
| 413  | Request entity too large (file upload).           |
| 429  | Rate limited.                                     |
| 503  | Service degraded (database unreachable, etc.).    |

---

## Rate Limiting

All API endpoints are subject to rate limiting to prevent abuse.

**Default Limits:**

| Scope          | Limit            | Window   |
|----------------|------------------|----------|
| Per session    | 300 requests     | 1 minute |
| Login endpoint | 10 attempts      | 15 minutes |

**Rate Limit Headers:**

All responses include the following headers:

| Header                  | Description                                          |
|-------------------------|------------------------------------------------------|
| `X-RateLimit-Limit`     | Maximum number of requests allowed in the window.    |
| `X-RateLimit-Remaining` | Number of requests remaining in the current window.  |
| `X-RateLimit-Reset`     | Unix timestamp when the rate limit window resets.    |
| `Retry-After`           | Seconds until the client can retry (only on 429).    |

**Rate Limit Exceeded Response (429):**

```json
{
  "error": "Rate limit exceeded"
}
```
