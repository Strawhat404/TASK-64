# Test Coverage Audit

## Project Type Detection
- README-declared project type: `fullstack` (`repo/README.md:3`).
- File-system confirmation: both backend (`repo/backend`) and frontend (`repo/frontend`) present.

## Backend Endpoint Inventory
- Total unique endpoints: **80** (`METHOD + PATH`) from `repo/backend/cmd/main.go` route registration.
| Endpoint | Source |
|---|---|
| `GET /api/health` | `repo/backend/cmd/main.go:90` |
| `POST /api/auth/login` | `repo/backend/cmd/main.go:99` |
| `POST /api/auth/logout` | `repo/backend/cmd/main.go:100` |
| `GET /api/auth/session` | `repo/backend/cmd/main.go:101` |
| `GET /api/users` | `repo/backend/cmd/main.go:108` |
| `GET /api/users/:id` | `repo/backend/cmd/main.go:109` |
| `POST /api/users` | `repo/backend/cmd/main.go:110` |
| `PUT /api/users/:id` | `repo/backend/cmd/main.go:111` |
| `DELETE /api/users/:id` | `repo/backend/cmd/main.go:112` |
| `GET /api/services` | `repo/backend/cmd/main.go:116` |
| `GET /api/services/:id` | `repo/backend/cmd/main.go:117` |
| `GET /api/services/:id/pricing` | `repo/backend/cmd/main.go:118` |
| `POST /api/services` | `repo/backend/cmd/main.go:119` |
| `PUT /api/services/:id` | `repo/backend/cmd/main.go:120` |
| `DELETE /api/services/:id` | `repo/backend/cmd/main.go:121` |
| `GET /api/schedules` | `repo/backend/cmd/main.go:125` |
| `POST /api/schedules` | `repo/backend/cmd/main.go:126` |
| `PUT /api/schedules/:id` | `repo/backend/cmd/main.go:127` |
| `DELETE /api/schedules/:id` | `repo/backend/cmd/main.go:128` |
| `POST /api/schedules/:id/confirm` | `repo/backend/cmd/main.go:129` |
| `GET /api/schedules/available-staff` | `repo/backend/cmd/main.go:130` |
| `POST /api/schedules/:id/backup` | `repo/backend/cmd/main.go:131` |
| `POST /api/schedules/backup/:id/confirm` | `repo/backend/cmd/main.go:132` |
| `GET /api/staff` | `repo/backend/cmd/main.go:136` |
| `GET /api/staff/:id` | `repo/backend/cmd/main.go:137` |
| `POST /api/staff` | `repo/backend/cmd/main.go:138` |
| `PUT /api/staff/:id` | `repo/backend/cmd/main.go:139` |
| `DELETE /api/staff/:id` | `repo/backend/cmd/main.go:140` |
| `GET /api/staff/:id/credentials` | `repo/backend/cmd/main.go:141` |
| `POST /api/staff/:id/credentials` | `repo/backend/cmd/main.go:142` |
| `GET /api/staff/:id/availability` | `repo/backend/cmd/main.go:143` |
| `POST /api/staff/:id/availability` | `repo/backend/cmd/main.go:144` |
| `GET /api/audit/logs` | `repo/backend/cmd/main.go:148` |
| `POST /api/governance/content` | `repo/backend/cmd/main.go:152` |
| `GET /api/governance/content` | `repo/backend/cmd/main.go:153` |
| `GET /api/governance/content/:id` | `repo/backend/cmd/main.go:154` |
| `PUT /api/governance/content/:id` | `repo/backend/cmd/main.go:155` |
| `POST /api/governance/content/:id/submit` | `repo/backend/cmd/main.go:156` |
| `POST /api/governance/content/:id/promote` | `repo/backend/cmd/main.go:157` |
| `GET /api/governance/content/:id/versions` | `repo/backend/cmd/main.go:158` |
| `POST /api/governance/content/:id/rollback` | `repo/backend/cmd/main.go:159` |
| `GET /api/governance/reviews/pending` | `repo/backend/cmd/main.go:160` |
| `POST /api/governance/reviews/:id/decide` | `repo/backend/cmd/main.go:161` |
| `GET /api/governance/gray-release` | `repo/backend/cmd/main.go:162` |
| `GET /api/governance/rules` | `repo/backend/cmd/main.go:163` |
| `POST /api/governance/rules` | `repo/backend/cmd/main.go:164` |
| `PUT /api/governance/rules/:id` | `repo/backend/cmd/main.go:165` |
| `DELETE /api/governance/rules/:id` | `repo/backend/cmd/main.go:166` |
| `GET /api/governance/content/:id/versions/diff` | `repo/backend/cmd/main.go:167` |
| `POST /api/governance/relationships` | `repo/backend/cmd/main.go:168` |
| `GET /api/governance/relationships` | `repo/backend/cmd/main.go:169` |
| `DELETE /api/governance/relationships/:id` | `repo/backend/cmd/main.go:170` |
| `POST /api/governance/content/:id/re-review` | `repo/backend/cmd/main.go:171` |
| `POST /api/reconciliation/import` | `repo/backend/cmd/main.go:175` |
| `GET /api/reconciliation/feeds` | `repo/backend/cmd/main.go:176` |
| `GET /api/reconciliation/feeds/:id` | `repo/backend/cmd/main.go:177` |
| `POST /api/reconciliation/feeds/:id/match` | `repo/backend/cmd/main.go:178` |
| `GET /api/reconciliation/matches` | `repo/backend/cmd/main.go:179` |
| `GET /api/reconciliation/exceptions` | `repo/backend/cmd/main.go:180` |
| `GET /api/reconciliation/exceptions/export` | `repo/backend/cmd/main.go:181` |
| `GET /api/reconciliation/exceptions/:id` | `repo/backend/cmd/main.go:182` |
| `PUT /api/reconciliation/exceptions/:id/assign` | `repo/backend/cmd/main.go:183` |
| `PUT /api/reconciliation/exceptions/:id/resolve` | `repo/backend/cmd/main.go:184` |
| `GET /api/reconciliation/summary` | `repo/backend/cmd/main.go:185` |
| `POST /api/security/sensitive` | `repo/backend/cmd/main.go:189` |
| `GET /api/security/sensitive` | `repo/backend/cmd/main.go:190` |
| `GET /api/security/sensitive/:id` | `repo/backend/cmd/main.go:191` |
| `POST /api/security/sensitive/:id/reveal` | `repo/backend/cmd/main.go:192` |
| `DELETE /api/security/sensitive/:id` | `repo/backend/cmd/main.go:193` |
| `POST /api/security/keys/rotate` | `repo/backend/cmd/main.go:194` |
| `GET /api/security/keys` | `repo/backend/cmd/main.go:195` |
| `GET /api/security/keys/rotation-due` | `repo/backend/cmd/main.go:196` |
| `GET /api/security/audit-ledger` | `repo/backend/cmd/main.go:197` |
| `POST /api/security/audit-ledger/verify` | `repo/backend/cmd/main.go:198` |
| `GET /api/security/retention` | `repo/backend/cmd/main.go:199` |
| `POST /api/security/retention/cleanup` | `repo/backend/cmd/main.go:200` |
| `GET /api/security/rate-limits` | `repo/backend/cmd/main.go:201` |
| `POST /api/security/legal-holds` | `repo/backend/cmd/main.go:202` |
| `GET /api/security/legal-holds` | `repo/backend/cmd/main.go:203` |
| `PUT /api/security/legal-holds/:id/release` | `repo/backend/cmd/main.go:204` |

## API Test Mapping Table
| Endpoint | Covered | Test Type | Test Files | Evidence |
|---|---|---|---|---|
| `GET /api/health` | yes | true no-mock HTTP | `repo/API_tests/api_test.sh, repo/tests/health_check.sh` | `repo/API_tests/api_test.sh:162` |
| `POST /api/auth/login` | yes | true no-mock HTTP | `repo/API_tests/api_test.sh, repo/API_tests/authz_boundary_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/api_test.sh:169` |
| `POST /api/auth/logout` | yes | true no-mock HTTP | `repo/API_tests/api_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/api_test.sh:497` |
| `GET /api/auth/session` | yes | true no-mock HTTP | `repo/API_tests/api_test.sh, repo/API_tests/authz_boundary_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/api_test.sh:190` |
| `GET /api/users` | yes | true no-mock HTTP | `repo/API_tests/api_test.sh, repo/API_tests/authz_boundary_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/api_test.sh:212` |
| `GET /api/users/:id` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:428` |
| `POST /api/users` | yes | true no-mock HTTP | `repo/API_tests/api_test.sh, repo/API_tests/authz_boundary_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/api_test.sh:202` |
| `PUT /api/users/:id` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:433` |
| `DELETE /api/users/:id` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:436` |
| `GET /api/services` | yes | true no-mock HTTP | `repo/API_tests/authz_boundary_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/authz_boundary_test.sh:357` |
| `GET /api/services/:id` | yes | true no-mock HTTP | `repo/API_tests/authz_boundary_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/authz_boundary_test.sh:192` |
| `GET /api/services/:id/pricing` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:460` |
| `POST /api/services` | yes | true no-mock HTTP | `repo/API_tests/api_test.sh, repo/API_tests/authz_boundary_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/api_test.sh:228` |
| `PUT /api/services/:id` | yes | true no-mock HTTP | `repo/API_tests/authz_boundary_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/authz_boundary_test.sh:332` |
| `DELETE /api/services/:id` | yes | true no-mock HTTP | `repo/API_tests/authz_boundary_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/authz_boundary_test.sh:338` |
| `GET /api/schedules` | yes | true no-mock HTTP | `repo/API_tests/authz_boundary_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/authz_boundary_test.sh:48` |
| `POST /api/schedules` | yes | true no-mock HTTP | `repo/API_tests/api_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/api_test.sh:275` |
| `PUT /api/schedules/:id` | yes | true no-mock HTTP | `repo/API_tests/authz_boundary_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/authz_boundary_test.sh:453` |
| `DELETE /api/schedules/:id` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:207` |
| `POST /api/schedules/:id/confirm` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:181` |
| `GET /api/schedules/available-staff` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:164` |
| `POST /api/schedules/:id/backup` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:188` |
| `POST /api/schedules/backup/:id/confirm` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:194` |
| `GET /api/staff` | yes | true no-mock HTTP | `repo/API_tests/api_test.sh, repo/API_tests/authz_boundary_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/api_test.sh:242` |
| `GET /api/staff/:id` | yes | true no-mock HTTP | `repo/API_tests/authz_boundary_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/authz_boundary_test.sh:410` |
| `POST /api/staff` | yes | true no-mock HTTP | `repo/API_tests/api_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/api_test.sh:263` |
| `PUT /api/staff/:id` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:133` |
| `DELETE /api/staff/:id` | yes | true no-mock HTTP | `repo/API_tests/authz_boundary_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/authz_boundary_test.sh:424` |
| `GET /api/staff/:id/credentials` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:141` |
| `POST /api/staff/:id/credentials` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:137` |
| `GET /api/staff/:id/availability` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:149` |
| `POST /api/staff/:id/availability` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:145` |
| `GET /api/audit/logs` | yes | true no-mock HTTP | `repo/API_tests/api_test.sh, repo/API_tests/authz_boundary_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/api_test.sh:346` |
| `POST /api/governance/content` | yes | true no-mock HTTP | `repo/API_tests/api_test.sh, repo/API_tests/authz_boundary_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/api_test.sh:374` |
| `GET /api/governance/content` | yes | true no-mock HTTP | `repo/API_tests/api_test.sh, repo/API_tests/authz_boundary_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/api_test.sh:389` |
| `GET /api/governance/content/:id` | yes | true no-mock HTTP | `repo/API_tests/authz_boundary_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/authz_boundary_test.sh:191` |
| `PUT /api/governance/content/:id` | yes | true no-mock HTTP | `repo/API_tests/authz_boundary_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/authz_boundary_test.sh:344` |
| `POST /api/governance/content/:id/submit` | yes | true no-mock HTTP | `repo/API_tests/api_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/api_test.sh:410` |
| `POST /api/governance/content/:id/promote` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:268` |
| `GET /api/governance/content/:id/versions` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:240` |
| `POST /api/governance/content/:id/rollback` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:272` |
| `GET /api/governance/reviews/pending` | yes | true no-mock HTTP | `repo/API_tests/api_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/api_test.sh:423` |
| `POST /api/governance/reviews/:id/decide` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:253` |
| `GET /api/governance/gray-release` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:103` |
| `GET /api/governance/rules` | yes | true no-mock HTTP | `repo/API_tests/api_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/api_test.sh:429` |
| `POST /api/governance/rules` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:281` |
| `PUT /api/governance/rules/:id` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:286` |
| `DELETE /api/governance/rules/:id` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:289` |
| `GET /api/governance/content/:id/versions/diff` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:243` |
| `POST /api/governance/relationships` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:302` |
| `GET /api/governance/relationships` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:103` |
| `DELETE /api/governance/relationships/:id` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:307` |
| `POST /api/governance/content/:id/re-review` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:259` |
| `POST /api/reconciliation/import` | yes | true no-mock HTTP | `repo/API_tests/api_test.sh` | `repo/API_tests/api_test.sh:318` |
| `GET /api/reconciliation/feeds` | yes | true no-mock HTTP | `repo/API_tests/authz_boundary_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/authz_boundary_test.sh:52` |
| `GET /api/reconciliation/feeds/:id` | yes | true no-mock HTTP | `repo/API_tests/authz_boundary_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/authz_boundary_test.sh:411` |
| `POST /api/reconciliation/feeds/:id/match` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:330` |
| `GET /api/reconciliation/matches` | yes | true no-mock HTTP | `repo/API_tests/authz_boundary_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/authz_boundary_test.sh:55` |
| `GET /api/reconciliation/exceptions` | yes | true no-mock HTTP | `repo/API_tests/api_test.sh, repo/API_tests/authz_boundary_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/api_test.sh:334` |
| `GET /api/reconciliation/exceptions/export` | yes | true no-mock HTTP | `repo/API_tests/authz_boundary_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/authz_boundary_test.sh:171` |
| `GET /api/reconciliation/exceptions/:id` | yes | true no-mock HTTP | `repo/API_tests/authz_boundary_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/authz_boundary_test.sh:171` |
| `PUT /api/reconciliation/exceptions/:id/assign` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:348` |
| `PUT /api/reconciliation/exceptions/:id/resolve` | yes | true no-mock HTTP | `repo/API_tests/authz_boundary_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/authz_boundary_test.sh:427` |
| `GET /api/reconciliation/summary` | yes | true no-mock HTTP | `repo/API_tests/authz_boundary_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/authz_boundary_test.sh:53` |
| `POST /api/security/sensitive` | yes | true no-mock HTTP | `repo/API_tests/api_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/api_test.sh:435` |
| `GET /api/security/sensitive` | yes | true no-mock HTTP | `repo/API_tests/api_test.sh, repo/API_tests/authz_boundary_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/api_test.sh:445` |
| `GET /api/security/sensitive/:id` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:392` |
| `POST /api/security/sensitive/:id/reveal` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:396` |
| `DELETE /api/security/sensitive/:id` | yes | true no-mock HTTP | `repo/API_tests/authz_boundary_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/authz_boundary_test.sh:440` |
| `POST /api/security/keys/rotate` | yes | true no-mock HTTP | `repo/API_tests/authz_boundary_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/authz_boundary_test.sh:147` |
| `GET /api/security/keys` | yes | true no-mock HTTP | `repo/API_tests/api_test.sh, repo/API_tests/authz_boundary_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/api_test.sh:451` |
| `GET /api/security/keys/rotation-due` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:365` |
| `GET /api/security/audit-ledger` | yes | true no-mock HTTP | `repo/API_tests/api_test.sh, repo/API_tests/authz_boundary_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/api_test.sh:457` |
| `POST /api/security/audit-ledger/verify` | yes | true no-mock HTTP | `repo/API_tests/authz_boundary_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/authz_boundary_test.sh:470` |
| `GET /api/security/retention` | yes | true no-mock HTTP | `repo/API_tests/api_test.sh, repo/API_tests/authz_boundary_test.sh, repo/API_tests/full_coverage_test.sh` | `repo/API_tests/api_test.sh:463` |
| `POST /api/security/retention/cleanup` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:371` |
| `GET /api/security/rate-limits` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:103` |
| `POST /api/security/legal-holds` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:375` |
| `GET /api/security/legal-holds` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:103` |
| `PUT /api/security/legal-holds/:id/release` | yes | true no-mock HTTP | `repo/API_tests/full_coverage_test.sh` | `repo/API_tests/full_coverage_test.sh:380` |

## API Test Classification
1. True No-Mock HTTP
   - `repo/API_tests/api_test.sh`
   - `repo/API_tests/authz_boundary_test.sh`
   - `repo/API_tests/full_coverage_test.sh`
   - `repo/tests/health_check.sh`
   - Evidence: direct `curl`/`do_request` to live API URLs (examples: `repo/API_tests/api_test.sh:162`, `repo/API_tests/api_test.sh:318`, `repo/API_tests/full_coverage_test.sh:94`, `repo/tests/health_check.sh:41`).
2. HTTP with Mocking
   - None detected for API HTTP scripts.
3. Non-HTTP (unit/integration without full HTTP routing)
   - Handler/middleware tests using `e.NewContext` + direct function call (examples: `repo/backend/internal/handlers/auth_test.go:23`, `repo/backend/internal/middleware/auth_test.go:19`).
   - Service tests (`repo/backend/internal/services/*_test.go`) and standalone `repo/unit_tests/*.go`.

## Mock Detection
- Frontend mocking detected extensively (`vi.mock`), e.g. `repo/frontend/src/pages/StaffRoster.test.tsx:8`, `repo/frontend/src/App.test.tsx:7`, `repo/frontend/src/main.test.tsx:7`.
- Backend mock framework usage (`gomock`, `sqlmock`, `testify/mock`, `jest.mock`, `vi.mock`) not detected in backend/unit/API shell tests.
- Direct controller/middleware invocation without router observed (`e.NewContext`), so these are not true route-through-HTTP tests.

## Coverage Summary
- Total endpoints: **80**
- Endpoints with HTTP tests: **80**
- Endpoints with TRUE no-mock tests: **80**
- HTTP coverage %: **100.0%**
- True API coverage %: **100.0%**

## Unit Test Summary
### Backend Unit Tests
- Test files include:
  - `repo/backend/internal/middleware/auth_test.go`
  - `repo/backend/internal/middleware/ratelimit_test.go`
  - `repo/backend/internal/middleware/security_test.go`
  - `repo/backend/internal/services/encryption_test.go`
  - `repo/backend/internal/services/reconciliation_test.go`
  - `repo/backend/internal/services/scheduler_test.go`
  - `repo/unit_tests/audit_hash_chain_test.go`
  - `repo/unit_tests/auth_helpers_test.go`
  - `repo/unit_tests/content_lifecycle_test.go`
  - `repo/unit_tests/cookie_security_test.go`
  - `repo/unit_tests/date_parsing_test.go`
  - `repo/unit_tests/encryption_comprehensive_test.go`
  - `repo/unit_tests/governance_workflow_test.go`
  - `repo/unit_tests/levenshtein_similarity_test.go`
  - `repo/unit_tests/masking_test.go`
  - `repo/unit_tests/match_scoring_comprehensive_test.go`
  - `repo/unit_tests/middleware_security_test.go`
  - `repo/unit_tests/model_validation_test.go`
  - `repo/unit_tests/password_validation_test.go`
  - `repo/unit_tests/pricing_calculation_test.go`
  - `repo/unit_tests/reconciliation_extended_test.go`
  - `repo/unit_tests/reconciliation_variance_test.go`
  - `repo/unit_tests/retention_legal_hold_test.go`
  - `repo/unit_tests/scheduling_buffer_test.go`
  - `repo/unit_tests/variance_severity_test.go`
- Modules covered:
  - Middleware/auth/guards/rate-limit constants and behavior checks (`repo/backend/internal/middleware/*_test.go`).
  - Services encryption/reconciliation/scheduler (`repo/backend/internal/services/*_test.go`).
  - Additional business-rule spec tests in `repo/unit_tests/*.go`.
- Important backend modules NOT directly unit-tested in `backend/internal` package tests:
  - `repo/backend/internal/models/models.go`
  - `repo/backend/internal/services/auditledger.go`
  - `repo/backend/internal/services/grayrelease.go`
  - `repo/backend/internal/services/moderation.go`
  - `repo/backend/internal/services/versioning.go`

### Frontend Unit Tests (STRICT REQUIREMENT)
- Frontend test files detected: **19**.
- Framework/tools evidence: `vitest`, `@testing-library/react`, `@testing-library/jest-dom` in `repo/frontend/package.json`; direct `render(...)` in test files.
- Components/modules covered include pages, context, components, and now app/bootstrap tests (`repo/frontend/src/App.test.tsx`, `repo/frontend/src/main.test.tsx`).
- Important frontend modules NOT tested: none identified among `.tsx` modules.
- **Frontend unit tests: PRESENT**

### Cross-Layer Observation
- Backend API coverage is comprehensive; frontend has substantial unit tests.
- Real browser-level FE↔BE E2E flow tests are still not evidenced in repository test suite.

## API Observability Check
- Endpoint visibility: strong (`do_request METHOD "/path"` and explicit curl targets).
- Request input visibility: strong (JSON payloads and multipart examples visible in scripts, e.g. `repo/API_tests/api_test.sh:318`).
- Response assertions: mixed.
  - Strong in `full_coverage_test.sh` (single expected statuses, explicit body field checks).
  - Weaker in `api_test.sh`/`authz_boundary_test.sh` where some assertions allow multiple outcomes or fallback paths.

## Tests Check
- Success paths: covered across all domains (auth, users, services, schedules, governance, reconciliation, security).
- Failure and permissions paths: covered (401/403/404 and invalid input paths).
- Edge cases: covered (buffer windows, scoring thresholds, cryptographic behavior, lifecycle transitions).
- Integration boundaries: API scripts hit real HTTP endpoints.
- Assertion depth: improved in strict full coverage script, but uneven across scripts.
- `run_tests.sh`: Docker-based execution confirmed; however helper container installs packages at runtime via `apk add` (`repo/run_tests.sh:118`).

## Test Coverage Score (0-100)
- **90/100**

## Score Rationale
- Very high endpoint coverage (80/80 true HTTP route hits).
- Broad unit/spec coverage across backend and frontend.
- Deductions for:
  - Lack of explicit FE↔BE end-to-end browser tests.
  - Some scripts still use permissive/fallback assertion patterns.
  - Several core backend files remain untested by direct package-level unit tests (notably models and some governance/security services).

## Key Gaps
- Missing explicit end-to-end UI-to-API automation for fullstack behavior.
- Important backend modules with no direct tests:
  - `repo/backend/internal/models/models.go`
  - `repo/backend/internal/models/governance.go`
  - `repo/backend/internal/models/reconciliation.go`
  - `repo/backend/internal/models/security.go`
  - `repo/backend/internal/services/auditledger.go`
  - `repo/backend/internal/services/grayrelease.go`
  - `repo/backend/internal/services/moderation.go`
  - `repo/backend/internal/services/versioning.go`

## Confidence & Assumptions
- Confidence: **High** for static route/test mapping and README gate checks.
- Assumptions:
  - Static inspection only; tests were not executed.
  - Coverage is inferred from declared test calls and route definitions in current files only.
  - Dynamic IDs/variables in scripts represent intended route invocation against real handlers.

# README Audit

## High Priority Issues
- None that violate README hard-gate requirements for fullstack startup/access/verification/credentials.
- Operational caveat: runtime package install appears in `run_tests.sh` helper container (`apk add`), which may be undesirable in restricted environments.

## Medium Priority Issues
- README does not explicitly warn that some API tests include permissive outcomes in non-strict scripts (`api_test.sh`, `authz_boundary_test.sh`).
- Verification section is strong for API/browser checks but does not include explicit FE automated test command path (`frontend` vitest) alongside unified runner.

## Low Priority Issues
- Very long endpoint table; readability could improve with grouped subsections by domain.
- Minor: could include explicit container health check command in Quick Start flow for deterministic operator checks.

## Hard Gate Failures
- None.
- `repo/README.md` exists at required path.
- Project type declared at top (`fullstack`).
- Startup includes `docker-compose up`.
- Access method includes URL+port (`https://localhost:3443`).
- Verification includes API curl and web UI flow.
- Demo credentials include all roles.

## README Verdict (PASS / PARTIAL PASS / FAIL)
- **PASS**

## Final Verdicts
- **Test Coverage Audit Verdict:** PARTIAL PASS (excellent endpoint coverage; remaining depth/E2E gaps).
- **README Audit Verdict:** PASS.
