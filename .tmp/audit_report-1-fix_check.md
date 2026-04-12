# Inspection Re-check Report (Rerun 2, 2026-04-12)

## Scope
Re-validated the same four previously reported issues after the latest round of fixes.

## Results

### 1) Security dashboard expects `broken_at`, API returns `broken_at_entry`
- Status: **Fixed**
- Evidence:
  - Frontend expects/uses `broken_at`: `repo/frontend/src/pages/SecurityDashboard.tsx:51-55`, `repo/frontend/src/pages/SecurityDashboard.tsx:315`
  - Backend now returns `broken_at` on invalid chain: `repo/backend/internal/handlers/security.go:407-409`

### 2) Documentation diverges from actual deployment/config
- Status: **Fixed**
- Evidence:
  - Design doc deployment model aligns with compose (4 services, startup sequence, init scripts): `docs/design.md:398-418`, `repo/docker-compose.yml:3-69`
  - README route/config sections no longer contain prior mismatched entries from the earlier report scope: `repo/README.md:56-92`
  - `.env.example` now reflects core DB/backend/tls values without the previously flagged extra port/domain block mismatch: `repo/.env.example:8-29`

### 3) README route matrix includes non-existent top-level route groups
- Status: **Fixed**
- Evidence:
  - README route matrix now lists only actual top-level groups: `repo/README.md:56-68`
  - Backend router confirms nested relationship/backup/credential/availability routes under existing groups: `repo/backend/cmd/main.go:123-171`

### 4) Authz boundary test relies on hardcoded role id and weak fallback that weakens strict 403 checks
- Status: **Fixed**
- Evidence:
  - Scheduler role ID is resolved dynamically (no hardcoded `role_id=2`): `repo/tests/API_tests/authz_boundary_test.sh:80-92`, `repo/tests/API_tests/authz_boundary_test.sh:102`
  - Role-boundary assertions explicitly require `403` for authenticated non-admin flow: `repo/tests/API_tests/authz_boundary_test.sh:128-142`, `repo/tests/API_tests/authz_boundary_test.sh:153-164`
  - Prior fallback that downgraded role-boundary expectations to `401` is removed.

## Overall conclusion
- **4 / 4 fixed**
