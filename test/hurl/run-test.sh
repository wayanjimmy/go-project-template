#!/usr/bin/env bash

set -u

BASE_URL="${BASE_URL:-http://localhost:3030}"
TEST_ROOT="${TEST_ROOT:-test/hurl}"
TESTS_FAILED=0

if ! command -v hurl >/dev/null 2>&1; then
  echo "❌ hurl is not installed or not in PATH"
  exit 1
fi

if [ ! -d "$TEST_ROOT" ]; then
  echo "❌ test directory not found: $TEST_ROOT"
  exit 1
fi

mapfile -t TEST_FILES < <(find "$TEST_ROOT" -type f -name "*.hurl" | sort)

if [ "${#TEST_FILES[@]}" -eq 0 ]; then
  echo "❌ no .hurl files found under $TEST_ROOT"
  exit 1
fi

declare -a ALL_TEST_NAMES
declare -a ALL_TEST_STATUSES

echo "Running ${#TEST_FILES[@]} Hurl test file(s)"
echo "BASE_URL=$BASE_URL"
echo

for test_file in "${TEST_FILES[@]}"; do
  test_name=$(grep -m1 '^# Test:' "$test_file" | cut -d':' -f2- | xargs)
  if [ -z "$test_name" ]; then
    test_name="$test_file"
  fi

  timestamp=$(date +%s)
  run_token=$(LC_ALL=C tr -dc 'a-z' </dev/urandom | head -c 10)
  run_id="${timestamp}-${run_token}"

  echo "▶ $test_name"
  hurl --test --continue-on-error --very-verbose \
    --retry 5 --retry-interval 500 \
    --variable "base_url=$BASE_URL" \
    --variable "run_id=$run_id" \
    --variable "run_token=$run_token" \
    "$test_file"

  exit_code=$?
  if [ "$exit_code" -eq 0 ]; then
    status="✅ Passed"
  else
    status="❌ Failed"
    TESTS_FAILED=1
  fi

  ALL_TEST_NAMES+=("$test_name")
  ALL_TEST_STATUSES+=("$status")
  echo
 done

echo "Final Test Summary"
echo "=================="
echo "Total Tests: ${#ALL_TEST_NAMES[@]}"

passed=0
failed=0
for status in "${ALL_TEST_STATUSES[@]}"; do
  if [ "$status" = "✅ Passed" ]; then
    passed=$((passed + 1))
  else
    failed=$((failed + 1))
  fi
 done

echo "Passed: $passed"
echo "Failed: $failed"
echo
echo "Detailed Results:"
echo "----------------"

for i in "${!ALL_TEST_NAMES[@]}"; do
  echo "${ALL_TEST_STATUSES[$i]}: ${ALL_TEST_NAMES[$i]}"
done

exit "$TESTS_FAILED"
