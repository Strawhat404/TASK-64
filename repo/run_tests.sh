#!/usr/bin/env bash
set -uo pipefail

###############################################################################
# Unified Test Runner
#
# Runs all Go unit tests and API/integration shell tests from inside Docker.
# No arguments required. Assumes `docker compose up -d` has been run.
#
# Usage:
#   ./run_tests.sh
###############################################################################

REPO_DIR="$(cd "$(dirname "$0")" && pwd)"
PASS=0
FAIL=0
TOTAL=0
RESULTS=()

# Colours (disabled when stdout is not a terminal)
if [ -t 1 ]; then
  GREEN='\033[0;32m'
  RED='\033[0;31m'
  BOLD='\033[1m'
  RESET='\033[0m'
else
  GREEN='' RED='' BOLD='' RESET=''
fi

record_pass() {
  TOTAL=$((TOTAL + 1))
  PASS=$((PASS + 1))
  RESULTS+=("${GREEN}PASS${RESET}  $1")
  echo -e "  ${GREEN}PASS${RESET}  $1"
}

record_fail() {
  TOTAL=$((TOTAL + 1))
  FAIL=$((FAIL + 1))
  RESULTS+=("${RED}FAIL${RESET}  $1")
  echo -e "  ${RED}FAIL${RESET}  $1"
}

###############################################################################
# Identify containers
###############################################################################
BACKEND_CONTAINER=$(docker ps --filter "name=backend" --filter "status=running" \
  --format '{{.Names}}' | head -1)

if [ -z "$BACKEND_CONTAINER" ]; then
  echo "ERROR: Backend container is not running."
  echo "       Start the stack first:  docker compose up -d"
  exit 1
fi

###############################################################################
# 1. Go Unit Tests (run inside the backend build image)
###############################################################################
echo ""
echo -e "${BOLD}=== Go Unit Tests (repo/unit_tests/) ===${RESET}"
echo ""

# We run go test inside a golang container that mounts the repo so it has
# access to go.mod (in backend/) and the unit test source files.
# The unit_tests package is standalone (stdlib only), so we initialise a
# temporary module for it.
UNIT_OUTPUT=$(docker run --rm \
  -v "$REPO_DIR/unit_tests:/src/unit_tests" \
  -w /src/unit_tests \
  golang:1.22-alpine sh -c '
    go mod init unit_tests 2>/dev/null
    go test -v -count=1 ./... 2>&1
  ' 2>&1)
UNIT_EXIT=$?

# Parse individual test results from go test -v output
while IFS= read -r line; do
  case "$line" in
    *"--- PASS:"*)
      test_name=$(echo "$line" | sed 's/.*--- PASS: \([^ ]*\).*/\1/')
      record_pass "unit: $test_name"
      ;;
    *"--- FAIL:"*)
      test_name=$(echo "$line" | sed 's/.*--- FAIL: \([^ ]*\).*/\1/')
      record_fail "unit: $test_name"
      ;;
  esac
done <<< "$UNIT_OUTPUT"

# If go test failed but we found no individual results, record a top-level failure
if [ "$UNIT_EXIT" -ne 0 ] && ! echo "$UNIT_OUTPUT" | grep -q '--- FAIL:'; then
  record_fail "unit: go test exited with code $UNIT_EXIT"
  echo "$UNIT_OUTPUT" | tail -20
fi

# If we got zero results (e.g. no test files found), note it
if ! echo "$UNIT_OUTPUT" | grep -q -- '--- PASS:' && ! echo "$UNIT_OUTPUT" | grep -q -- '--- FAIL:'; then
  if [ "$UNIT_EXIT" -eq 0 ]; then
    echo "  (no individual test results found)"
  fi
fi

###############################################################################
# 2. API / Integration Tests (shell scripts in repo/API_tests/)
###############################################################################
echo ""
echo -e "${BOLD}=== API / Integration Tests (repo/API_tests/) ===${RESET}"
echo ""

for script in "$REPO_DIR"/API_tests/*_test.sh; do
  [ -f "$script" ] || continue
  script_name="$(basename "$script")"
  echo "--- Running: $script_name ---"

  # Run inside the backend container's network via a helper container that
  # has curl, bash, python3, and can reach localhost:8080 through Docker
  # networking. We exec into the backend container's network namespace.
  OUTPUT=$(docker run --rm \
    --network container:"$BACKEND_CONTAINER" \
    -v "$REPO_DIR/API_tests:/tests" \
    alpine:3.20 sh -c '
      apk add --no-cache -q bash curl python3 docker-cli >/dev/null 2>&1
      chmod +x /tests/'"$script_name"'
      bash /tests/'"$script_name"' 2>&1
    ' 2>&1)
  EXIT_CODE=$?

  if [ $EXIT_CODE -eq 0 ]; then
    record_pass "api: $script_name"
  else
    record_fail "api: $script_name"
  fi

  # Print the script's own output indented
  echo "$OUTPUT" | sed 's/^/    /'
  echo ""
done

###############################################################################
# 3. Summary
###############################################################################
echo ""
echo -e "${BOLD}============================================${RESET}"
echo -e "${BOLD}Test Summary${RESET}"
echo -e "${BOLD}============================================${RESET}"
echo ""

for r in "${RESULTS[@]}"; do
  echo -e "  $r"
done

echo ""
echo -e "  Total:  $TOTAL"
echo -e "  Passed: ${GREEN}$PASS${RESET}"
echo -e "  Failed: ${RED}$FAIL${RESET}"
echo ""

if [ "$FAIL" -gt 0 ]; then
  echo -e "${RED}Some tests failed.${RESET}"
  exit 1
else
  echo -e "${GREEN}All tests passed.${RESET}"
  exit 0
fi
