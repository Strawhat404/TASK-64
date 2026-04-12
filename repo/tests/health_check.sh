#!/usr/bin/env bash
set -euo pipefail

BASE_URL="https://localhost:3443"
CURL_OPTS="-s -k --max-time 5"
MAX_RETRIES=30
RETRY_INTERVAL=2

PASS=0
FAIL=0

pass() {
    echo "PASS: $1"
    PASS=$((PASS + 1))
}

fail() {
    echo "FAIL: $1"
    FAIL=$((FAIL + 1))
}

# ------------------------------------------------------------------
# Wait for backend to become ready
# ------------------------------------------------------------------
echo "Waiting for backend to be ready..."
for i in $(seq 1 "$MAX_RETRIES"); do
    if curl $CURL_OPTS "${BASE_URL}/api/health" >/dev/null 2>&1; then
        echo "Backend is ready (attempt ${i}/${MAX_RETRIES})."
        break
    fi
    if [ "$i" -eq "$MAX_RETRIES" ]; then
        echo "Backend did not become ready after ${MAX_RETRIES} attempts."
        exit 1
    fi
    sleep "$RETRY_INTERVAL"
done

# ------------------------------------------------------------------
# Test 1: GET /api/health returns "ok"
# ------------------------------------------------------------------
HEALTH_RESPONSE=$(curl $CURL_OPTS "${BASE_URL}/api/health")
if echo "$HEALTH_RESPONSE" | grep -qi "ok"; then
    pass "GET /api/health contains 'ok'"
else
    fail "GET /api/health - expected 'ok', got: ${HEALTH_RESPONSE}"
fi

# ------------------------------------------------------------------
# Test 2: Frontend is reachable at /
# ------------------------------------------------------------------
HTTP_CODE=$(curl $CURL_OPTS -o /dev/null -w "%{http_code}" "${BASE_URL}/")
if [ "$HTTP_CODE" -eq 200 ]; then
    pass "GET / returned HTTP 200"
else
    fail "GET / returned HTTP ${HTTP_CODE}, expected 200"
fi

# ------------------------------------------------------------------
# Summary
# ------------------------------------------------------------------
echo ""
echo "=============================="
echo "Results: ${PASS} passed, ${FAIL} failed"
echo "=============================="

if [ "$FAIL" -gt 0 ]; then
    exit 1
fi

exit 0
