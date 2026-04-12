# Issue Recheck (Static) - 2026-04-12

## Scope
Re-verified the 4 issues you listed using static code/doc evidence only.

## Results

1. **High: Reconciliation matching not feed-scoped**
- **Status:** Fixed
- **Evidence:** `repo/backend/internal/services/reconciliation.go:18-27`, `repo/backend/internal/services/reconciliation.go:486-493`, `repo/backend/cmd/main.go:178`
- **Reason:** Matching now loads unmatched transactions with `feedID` filter (`tenant_id + feed_id + source + matched=false`).

2. **Fail: Authz test used wrong audit-ledger endpoint**
- **Status:** Fixed
- **Evidence:** `repo/tests/API_tests/authz_boundary_test.sh:449`, `repo/backend/cmd/main.go:198`
- **Reason:** Test now calls `POST /security/audit-ledger/verify`, matching the registered route.

3. **Partial Fail: High-risk authz/object-isolation tests were shallow**
- **Status:** Fixed (materially strengthened)
- **Evidence:** `repo/tests/API_tests/authz_boundary_test.sh:211-217`, `repo/tests/API_tests/authz_boundary_test.sh:226-235`, `repo/tests/API_tests/authz_boundary_test.sh:298-307`
- **Reason:** Script now includes fixture-driven cross-tenant setup and explicit cross-tenant read/mutation checks, not only fake-UUID/filtered-200 patterns.

4. **Partial Fail: TLS requirement softened by HTTP API base URL**
- **Status:** Fixed
- **Evidence:** `docs/api-spec.md:4`, `repo/README.md:18`, `repo/backend/cmd/main.go:217-225`
- **Reason:** API spec base URL is now HTTPS and is consistent with product docs/runtime TLS path.

## Final Recheck Verdict
- For these 4 cited issues: **All fixed** based on current static evidence.
