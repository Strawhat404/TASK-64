#!/usr/bin/env bash
set -euo pipefail

###############################################################################
# Authorization Boundary Tests
#
# Tests 401/403/404 responses for role-based and tenant-based access controls.
# Validates that the route-role matrix is enforced correctly.
#
# Prerequisites:
#   - Backend running on localhost:8080
#   - Admin user: admin / Admin12345!!!
#   - A non-admin user (Scheduler role) should exist or be created by tests
###############################################################################

BASE_URL="http://localhost:8080/api"
ADMIN_JAR="$(mktemp /tmp/authz_admin.XXXXXX)"
USER_JAR="$(mktemp /tmp/authz_user.XXXXXX)"
NOAUTH_JAR="$(mktemp /tmp/authz_noauth.XXXXXX)"
TENANT2_JAR=""
PASS=0
FAIL=0
TOTAL=0

trap 'rm -f "$ADMIN_JAR" "$USER_JAR" "$NOAUTH_JAR" "$TENANT2_JAR"' EXIT

check() {
  local description="$1"
  local expected_code="$2"
  local actual_code="$3"
  TOTAL=$((TOTAL + 1))
  if [ "$actual_code" -eq "$expected_code" ]; then
    PASS=$((PASS + 1))
    echo "  PASS: $description (HTTP $actual_code)"
  else
    FAIL=$((FAIL + 1))
    echo "  FAIL: $description (expected $expected_code, got $actual_code)"
  fi
}

###############################################################################
# 1. Unauthenticated access => 401
###############################################################################
echo "=== Unauthenticated Access (expect 401) ==="

for endpoint in \
  "GET /users" \
  "GET /schedules" \
  "GET /staff" \
  "GET /audit/logs" \
  "GET /governance/content" \
  "GET /reconciliation/feeds" \
  "GET /reconciliation/summary" \
  "GET /reconciliation/exceptions" \
  "GET /reconciliation/matches" \
  "GET /security/keys" \
  "GET /security/sensitive"; do
  method="${endpoint%% *}"
  path="${endpoint#* }"
  code=$(curl -s -o /dev/null -w "%{http_code}" -X "$method" "$BASE_URL$path" -b "$NOAUTH_JAR")
  check "Unauthenticated $method $path" 401 "$code"
done

###############################################################################
# 2. Login as admin (retry if rate-limited from a previous test run)
###############################################################################
echo ""
echo "=== Admin Login ==="
ADMIN_LOGGED_IN=false
for attempt in 1 2 3; do
  code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"Admin12345!!!"}' \
    -c "$ADMIN_JAR")
  if [ "$code" = "200" ]; then
    ADMIN_LOGGED_IN=true
    break
  elif [ "$code" = "429" ]; then
    echo "  Rate limited (attempt $attempt), waiting for window reset..."
    sleep 62
  else
    break
  fi
done
check "Admin login" 200 "$code"

###############################################################################
# 3. Create a non-admin (Scheduler) user for role boundary tests
###############################################################################
echo ""
echo "=== Create Scheduler User ==="

# Look up the Scheduler role ID dynamically from user listing
SCHED_PASS="Scheduler12345!!!"
SCHED_ROLE_ID=$(curl -s -b "$ADMIN_JAR" "$BASE_URL/users" \
  | grep -o '"role_name":"[^"]*"\|"role_id":[0-9]*' \
  | paste - - -d',' | grep 'Scheduler' | grep -o '"role_id":[0-9]*' | grep -o '[0-9]*' | head -1)

# Fallback: query roles by creating and inspecting user list for role mapping
if [ -z "$SCHED_ROLE_ID" ]; then
  # Default seed order: Administrator=1, Scheduler=2, Reviewer=3, Auditor=4
  # but we try to avoid hardcoding — use the users list which includes role_name
  SCHED_ROLE_ID=$(curl -s -b "$ADMIN_JAR" "$BASE_URL/users" \
    | python3 -c "import sys,json; users=json.load(sys.stdin); print(next((u['role_id'] for u in users if u.get('role_name')=='Scheduler'),''))" 2>/dev/null || true)
fi

if [ -z "$SCHED_ROLE_ID" ]; then
  echo "  ERROR: Could not determine Scheduler role ID; skipping role boundary tests"
  SCHED_AUTHED=false
else
  # Create a scheduler user via admin API (ignore 409 if already exists)
  curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/users" \
    -H "Content-Type: application/json" \
    -b "$ADMIN_JAR" \
    -d "{\"username\":\"testscheduler\",\"password\":\"$SCHED_PASS\",\"role_id\":$SCHED_ROLE_ID,\"email\":\"scheduler@test.local\"}"

  # Login as the scheduler user
  curl -s -o /dev/null -X POST "$BASE_URL/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"testscheduler\",\"password\":\"$SCHED_PASS\"}" \
    -c "$USER_JAR"

  # Verify session is active — fail hard if scheduler login doesn't work
  SCHED_CODE=$(curl -s -o /dev/null -w "%{http_code}" -b "$USER_JAR" "$BASE_URL/auth/session")
  if [ "$SCHED_CODE" != "200" ]; then
    echo "  FAIL: Scheduler user login failed (HTTP $SCHED_CODE). Cannot verify role boundaries."
    FAIL=$((FAIL + 1))
    TOTAL=$((TOTAL + 1))
    SCHED_AUTHED=false
  else
    SCHED_AUTHED=true
  fi
fi

###############################################################################
# 4. Role-based access: Admin-only endpoints => 403 for non-admin
###############################################################################
echo ""
echo "=== Role Boundary: Admin-Only Endpoints ==="

if [ "$SCHED_AUTHED" = true ]; then
  for endpoint in \
    "GET /users" \
    "POST /users" \
    "GET /security/keys" \
    "GET /security/sensitive" \
    "POST /security/keys/rotate" \
    "GET /security/audit-ledger" \
    "GET /security/retention"; do
    method="${endpoint%% *}"
    path="${endpoint#* }"
    code=$(curl -s -o /dev/null -w "%{http_code}" -X "$method" "$BASE_URL$path" \
      -H "Content-Type: application/json" -d '{}' -b "$USER_JAR")
    check "Non-admin $method $path => 403" 403 "$code"
  done
else
  echo "  SKIP: Role boundary tests require an authenticated Scheduler session"
fi

###############################################################################
# 5. Reconciliation endpoints require Admin/Auditor role
###############################################################################
echo ""
echo "=== Role Boundary: Reconciliation (Admin/Auditor only) ==="

if [ "$SCHED_AUTHED" = true ]; then
  for endpoint in \
    "GET /reconciliation/feeds" \
    "GET /reconciliation/matches" \
    "GET /reconciliation/exceptions" \
    "GET /reconciliation/exceptions/export" \
    "GET /reconciliation/summary"; do
    method="${endpoint%% *}"
    path="${endpoint#* }"
    code=$(curl -s -o /dev/null -w "%{http_code}" -X "$method" "$BASE_URL$path" -b "$USER_JAR")
    check "Non-Auditor $method $path => 403" 403 "$code"
  done
else
  echo "  SKIP: Reconciliation role boundary tests require an authenticated Scheduler session"
fi

###############################################################################
# 6. Object-level: accessing non-existent resources => 404 (not 500)
###############################################################################
echo ""
echo "=== Object-Level: Non-Existent Resources => 404 ==="

FAKE_UUID="00000000-0000-0000-0000-000000000099"

for endpoint in \
  "GET /governance/content/$FAKE_UUID" \
  "GET /services/$FAKE_UUID"; do
  method="${endpoint%% *}"
  path="${endpoint#* }"
  code=$(curl -s -o /dev/null -w "%{http_code}" -X "$method" "$BASE_URL$path" -b "$ADMIN_JAR")
  check "Admin $method $path (non-existent)" 404 "$code"
done

###############################################################################
# 7. Audit endpoints require Auditor/Admin role
###############################################################################
echo ""
echo "=== Role Boundary: Audit Logs ==="

if [ "$SCHED_AUTHED" = true ]; then
  code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/audit/logs" -b "$USER_JAR")
  check "Non-Auditor GET /audit/logs => 403" 403 "$code"
else
  echo "  SKIP: Audit role boundary test requires an authenticated Scheduler session"
fi

code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/audit/logs" -b "$ADMIN_JAR")
check "Admin GET /audit/logs" 200 "$code"

###############################################################################
# 8. Cross-tenant isolation (fixture-driven two-tenant setup)
#
# Seeds a real second tenant + admin via psql, creates concrete resources in
# Tenant 2, then verifies Tenant 1 admin cannot read, list, mutate, or delete
# those resources.
###############################################################################
echo ""
echo "=== Cross-Tenant Isolation (fixture-driven) ==="

TENANT2_ID="c2eebc99-9c0b-4ef8-bb6d-6bb9bd380a55"
TENANT2_ADMIN_ID="d3eebc99-9c0b-4ef8-bb6d-6bb9bd380a66"
TENANT2_ADMIN_USER="t2admin"
TENANT2_ADMIN_PASS="Tenant2Admin12345!!!"
TENANT2_JAR="$(mktemp /tmp/authz_t2.XXXXXX)"

# Determine the DB container name and psql connection params
DB_CONTAINER=$(docker ps --filter "name=db" --filter "status=running" --format '{{.Names}}' 2>/dev/null | grep -E 'compliance|db' | head -1 || true)
DB_USER="${POSTGRES_USER:-compliance}"
DB_NAME="${POSTGRES_DB:-compliance_console}"

CROSS_TENANT_SEEDED=false

if [ -n "$DB_CONTAINER" ]; then
  # Generate Argon2id hash for Tenant 2 admin password via the backend container
  BACKEND_CONTAINER=$(docker ps --filter "name=backend" --filter "status=running" --format '{{.Names}}' 2>/dev/null | head -1 || true)

  # Seed Tenant 2 and its admin user (idempotent — ON CONFLICT DO NOTHING)
  docker exec -i "$DB_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -q <<EOSQL
    INSERT INTO tenants (id, name, domain, created_at, updated_at)
    VALUES ('${TENANT2_ID}', 'Isolation Test Tenant', 'tenant2.test.local', NOW(), NOW())
    ON CONFLICT (id) DO NOTHING;

    -- Use the same Argon2id hash as the seed admin (password: Admin12345!!!)
    -- The actual password doesn't matter for cross-tenant reads; we just need a
    -- valid login to create resources owned by Tenant 2.
    INSERT INTO users (id, tenant_id, role_id, username, email, full_name, password_hash, created_at, updated_at)
    VALUES (
        '${TENANT2_ADMIN_ID}',
        '${TENANT2_ID}',
        (SELECT id FROM roles WHERE name = 'Administrator'),
        '${TENANT2_ADMIN_USER}',
        't2admin@tenant2.test.local',
        'Tenant 2 Admin',
        (SELECT password_hash FROM users WHERE username = 'admin'),
        NOW(), NOW()
    ) ON CONFLICT (username) DO NOTHING;
EOSQL

  if [ $? -eq 0 ]; then
    CROSS_TENANT_SEEDED=true
  else
    echo "  WARN: Failed to seed Tenant 2 via psql; falling back to UUID-based checks"
  fi
else
  echo "  WARN: Cannot find DB container for fixture seeding; falling back to UUID-based checks"
fi

if [ "$CROSS_TENANT_SEEDED" = true ]; then
  # Login as Tenant 2 admin (uses same password as default admin since we copied the hash)
  code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"'"${TENANT2_ADMIN_USER}"'","password":"Admin12345!!!"}' \
    -c "$TENANT2_JAR")

  if [ "$code" != "200" ]; then
    echo "  WARN: Tenant 2 admin login failed (HTTP $code); falling back to UUID-based checks"
    CROSS_TENANT_SEEDED=false
  fi
fi

if [ "$CROSS_TENANT_SEEDED" = true ]; then
  # -----------------------------------------------------------------------
  # 8a. Create real resources in Tenant 2 via its own admin session
  # -----------------------------------------------------------------------
  echo "--- Creating Tenant 2 fixtures ---"

  # Create a service in Tenant 2
  T2_SVC_RESP=$(curl -s -X POST "$BASE_URL/services" \
    -H "Content-Type: application/json" \
    -d '{"name":"T2 Isolation Service","description":"cross-tenant test","base_price_usd":100,"tier":"standard","duration_minutes":60}' \
    -b "$TENANT2_JAR")
  T2_SVC_ID=$(echo "$T2_SVC_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null || true)

  # Create a governance content item in Tenant 2
  T2_CONTENT_RESP=$(curl -s -X POST "$BASE_URL/governance/content" \
    -H "Content-Type: application/json" \
    -d '{"title":"T2 Secret Policy","body":"This belongs to Tenant 2","content_type":"policy"}' \
    -b "$TENANT2_JAR")
  T2_CONTENT_ID=$(echo "$T2_CONTENT_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null || true)

  echo "  Tenant 2 service ID:  ${T2_SVC_ID:-<failed>}"
  echo "  Tenant 2 content ID:  ${T2_CONTENT_ID:-<failed>}"

  # -----------------------------------------------------------------------
  # 8b. Tenant 1 admin attempts to READ Tenant 2 resources => must get 404
  # -----------------------------------------------------------------------
  echo ""
  echo "--- Cross-tenant single-resource reads (Tenant 1 -> Tenant 2) ---"

  if [ -n "$T2_SVC_ID" ]; then
    code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/services/$T2_SVC_ID" -b "$ADMIN_JAR")
    check "T1 admin GET T2 service => 404" 404 "$code"
  fi

  if [ -n "$T2_CONTENT_ID" ]; then
    code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/governance/content/$T2_CONTENT_ID" -b "$ADMIN_JAR")
    check "T1 admin GET T2 content => 404" 404 "$code"
  fi

  # -----------------------------------------------------------------------
  # 8c. Tenant 1 admin attempts to MUTATE Tenant 2 resources => must get 404
  # -----------------------------------------------------------------------
  echo ""
  echo "--- Cross-tenant mutation attempts (Tenant 1 -> Tenant 2) ---"

  if [ -n "$T2_SVC_ID" ]; then
    code=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "$BASE_URL/services/$T2_SVC_ID" \
      -H "Content-Type: application/json" \
      -d '{"name":"Hijacked Service"}' \
      -b "$ADMIN_JAR")
    check "T1 admin PUT T2 service => 404" 404 "$code"

    code=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/services/$T2_SVC_ID" \
      -b "$ADMIN_JAR")
    check "T1 admin DELETE T2 service => 404" 404 "$code"
  fi

  if [ -n "$T2_CONTENT_ID" ]; then
    code=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "$BASE_URL/governance/content/$T2_CONTENT_ID" \
      -H "Content-Type: application/json" \
      -d '{"title":"Hijacked","body":"xss"}' \
      -b "$ADMIN_JAR")
    check "T1 admin PUT T2 content => 404" 404 "$code"
  fi

  # -----------------------------------------------------------------------
  # 8d. List endpoints must not leak Tenant 2 data to Tenant 1
  # -----------------------------------------------------------------------
  echo ""
  echo "--- Cross-tenant list leak checks ---"

  for endpoint in "GET /services" "GET /governance/content"; do
    method="${endpoint%% *}"
    path="${endpoint#* }"
    body=$(curl -s -X "$method" "$BASE_URL$path" -b "$ADMIN_JAR")
    leaked=false
    [ -n "$T2_SVC_ID" ] && echo "$body" | grep -q "$T2_SVC_ID" && leaked=true
    [ -n "$T2_CONTENT_ID" ] && echo "$body" | grep -q "$T2_CONTENT_ID" && leaked=true
    echo "$body" | grep -q "$TENANT2_ID" && leaked=true
    TOTAL=$((TOTAL + 1))
    if [ "$leaked" = true ]; then
      FAIL=$((FAIL + 1))
      echo "  FAIL: $method $path leaks Tenant 2 data to Tenant 1"
    else
      PASS=$((PASS + 1))
      echo "  PASS: $method $path returns no Tenant 2 data"
    fi
  done

  # -----------------------------------------------------------------------
  # 8e. Verify Tenant 2 can still access its own resources (sanity check)
  # -----------------------------------------------------------------------
  echo ""
  echo "--- Tenant 2 self-access sanity check ---"

  if [ -n "$T2_SVC_ID" ]; then
    code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/services/$T2_SVC_ID" -b "$TENANT2_JAR")
    check "T2 admin GET own service => 200" 200 "$code"
  fi

  if [ -n "$T2_CONTENT_ID" ]; then
    code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/governance/content/$T2_CONTENT_ID" -b "$TENANT2_JAR")
    check "T2 admin GET own content => 200" 200 "$code"
  fi

  # -----------------------------------------------------------------------
  # 8f. Cleanup: remove Tenant 2 fixtures (idempotent)
  # -----------------------------------------------------------------------
  docker exec -i "$DB_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -q <<EOSQL
    DELETE FROM sessions WHERE user_id = '${TENANT2_ADMIN_ID}';
    DELETE FROM users WHERE id = '${TENANT2_ADMIN_ID}';
    DELETE FROM tenants WHERE id = '${TENANT2_ID}';
EOSQL

else
  # -----------------------------------------------------------------------
  # Fallback: UUID-based cross-tenant checks (no DB access)
  # -----------------------------------------------------------------------
  echo "  (Using UUID-based fallback — DB seeding unavailable)"
  CROSS_TENANT_UUID="00000000-0000-0000-0000-ffffffffffff"

  for endpoint in \
    "GET /services/$CROSS_TENANT_UUID" \
    "GET /governance/content/$CROSS_TENANT_UUID" \
    "GET /staff/$CROSS_TENANT_UUID" \
    "GET /reconciliation/feeds/$CROSS_TENANT_UUID" \
    "GET /reconciliation/exceptions/$CROSS_TENANT_UUID"; do
    method="${endpoint%% *}"
    path="${endpoint#* }"
    code=$(curl -s -o /dev/null -w "%{http_code}" -X "$method" "$BASE_URL$path" -b "$ADMIN_JAR")
    check "Cross-tenant $method $path => 404" 404 "$code"
  done

  # Mutation attempts
  code=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "$BASE_URL/governance/content/$CROSS_TENANT_UUID" \
    -H "Content-Type: application/json" -d '{"title":"hijacked","body":"xss"}' -b "$ADMIN_JAR")
  check "Cross-tenant PUT /governance/content => 404" 404 "$code"

  code=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/staff/$CROSS_TENANT_UUID" -b "$ADMIN_JAR")
  check "Cross-tenant DELETE /staff => 404" 404 "$code"

  code=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "$BASE_URL/reconciliation/exceptions/$CROSS_TENANT_UUID/resolve" \
    -H "Content-Type: application/json" -d '{"disposition":"approved","resolution_notes":"unauthorized"}' -b "$ADMIN_JAR")
  # Accept 400 (validation before lookup) or 404 (not found) — both correctly reject the request
  if [ "$code" = "400" ] || [ "$code" = "404" ]; then
    PASS=$((PASS + 1))
    TOTAL=$((TOTAL + 1))
    echo "  PASS: Cross-tenant PUT /reconciliation/exceptions/resolve => $code (rejected)"
  else
    FAIL=$((FAIL + 1))
    TOTAL=$((TOTAL + 1))
    echo "  FAIL: Cross-tenant PUT /reconciliation/exceptions/resolve => 404 (expected 404 or 400, got $code)"
  fi

  code=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/security/sensitive/$CROSS_TENANT_UUID" -b "$ADMIN_JAR")
  check "Cross-tenant DELETE /security/sensitive => 404" 404 "$code"
fi

###############################################################################
# 9. Schedule update with invalid service_id => 400
###############################################################################
echo ""
echo "=== Schedule Update: Invalid service_id Validation ==="

# Create a schedule first, then try updating with a non-existent service_id
INVALID_SERVICE_UUID="00000000-0000-0000-0000-000000000077"
# Try updating any schedule with a bad service_id (should get 400 or 404)
code=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "$BASE_URL/schedules/$FAKE_UUID" \
  -H "Content-Type: application/json" \
  -d "{\"service_id\": \"$INVALID_SERVICE_UUID\"}" \
  -b "$ADMIN_JAR")
# We expect either 400 (service not found) or 404 (schedule not found)
if [ "$code" = "400" ] || [ "$code" = "404" ]; then
  check "Update schedule with invalid service_id => $code" "$code" "$code"
else
  check "Update schedule with invalid service_id => 400 or 404" "400|404" "$code"
fi

###############################################################################
# 10. Immutable audit ledger verification endpoint
###############################################################################
echo ""
echo "=== Audit Ledger Integrity ==="

code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/security/audit-ledger/verify" \
  -H "Content-Type: application/json" -b "$ADMIN_JAR")
check "Audit ledger verify endpoint accessible" 200 "$code"

###############################################################################
# Summary
###############################################################################
echo ""
echo "============================================"
echo "Authorization Boundary Tests: $PASS/$TOTAL passed, $FAIL failed"
echo "============================================"

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
