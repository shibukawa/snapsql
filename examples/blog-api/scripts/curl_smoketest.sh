#!/usr/bin/env bash
set -euo pipefail

API_BASE_URL=${API_BASE_URL:-http://localhost:8000}
USER_SUFFIX=${USER_SUFFIX:-$(date +%s)}
USERNAME="smoketest-${USER_SUFFIX}"
USER_EMAIL="${USERNAME}@example.com"
TMP_DIR=$(mktemp -d)
trap 'rm -rf "${TMP_DIR}"' EXIT

ts() {
  date '+%Y-%m-%dT%H:%M:%S%z'
}

log() {
  printf '\n[%s] %s\n' "$(ts)" "$*"
}

run_curl() {
  local method=$1
  local path=$2
  shift 2
  log "${method} ${path}"
  curl -sS -X "${method}" "${API_BASE_URL}${path}" "$@"
  printf '\n'
}

log "Using base URL: ${API_BASE_URL}"

run_curl GET /health
run_curl GET /
run_curl GET /users/

log "POST /users/ (username=${USERNAME})"
USER_BODY=$(cat <<JSON
{
  "username": "${USERNAME}",
  "email": "${USER_EMAIL}",
  "full_name": "Smoke Tester",
  "bio": "Created via curl_smoketest.sh"
}
JSON
)
CREATE_RESP="${TMP_DIR}/user_create.json"
curl -sS -X POST "${API_BASE_URL}/users/" \
  -H 'Content-Type: application/json' \
  -d "${USER_BODY}" | tee "${CREATE_RESP}" | cat
printf '\n'
USER_ID=$(grep -o '"user_id":[0-9]*' "${CREATE_RESP}" | head -n1 | cut -d: -f2 || true)
if [[ -z "${USER_ID}" ]]; then
  echo "Failed to parse user_id from response" >&2
  exit 1
fi

run_curl GET "/users/${USER_ID}"

log "GET /posts/ (expected 501 until SnapSQL-generated handlers are wired in)"
run_curl GET /posts/

log "GET /comments/post/1 (expected 501 until handlers are implemented)"
run_curl GET /comments/post/1

log "Smoke test complete. Created user_id=${USER_ID}"
