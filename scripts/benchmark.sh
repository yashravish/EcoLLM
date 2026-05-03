#!/usr/bin/env bash
# Benchmark the EcoLLM API with a mix of simple/medium/hard prompts.
# Requires: wrk (https://github.com/wg/wrk) and curl.
set -euo pipefail

API_URL="${API_URL:-http://localhost:8080}"
API_KEY="${BENCHMARK_API_KEY:-}"
DURATION="${BENCHMARK_DURATION:-30s}"
THREADS="${BENCHMARK_THREADS:-4}"
CONNECTIONS="${BENCHMARK_CONNECTIONS:-20}"

if [ -z "$API_KEY" ]; then
  echo "ERROR: Set BENCHMARK_API_KEY to a valid API key" >&2
  exit 1
fi

SIMPLE_PAYLOAD='{"messages":[{"role":"user","content":"What is 2+2?"}],"max_tokens":50}'
MEDIUM_PAYLOAD='{"messages":[{"role":"user","content":"Explain the difference between TCP and UDP in 3 sentences."}],"max_tokens":150}'

echo "=== EcoLLM Benchmark ==="
echo "Target: $API_URL"
echo "Duration: $DURATION | Threads: $THREADS | Connections: $CONNECTIONS"
echo ""

# Health check first
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/health")
if [ "$HTTP_CODE" != "200" ]; then
  echo "ERROR: API health check failed (HTTP $HTTP_CODE)" >&2
  exit 1
fi

# Write a wrk Lua script for the completions endpoint
LUA_SCRIPT=$(mktemp /tmp/ecollm_bench_XXXXXX.lua)
cat > "$LUA_SCRIPT" << EOF
wrk.method = "POST"
wrk.headers["Content-Type"] = "application/json"
wrk.headers["Authorization"] = "Bearer $API_KEY"
wrk.body = '$SIMPLE_PAYLOAD'
EOF

echo "--- Simple prompts ---"
wrk -t"$THREADS" -c"$CONNECTIONS" -d"$DURATION" \
  --script="$LUA_SCRIPT" \
  "$API_URL/v1/chat/completions"

echo ""
cat > "$LUA_SCRIPT" << EOF
wrk.method = "POST"
wrk.headers["Content-Type"] = "application/json"
wrk.headers["Authorization"] = "Bearer $API_KEY"
wrk.body = '$MEDIUM_PAYLOAD'
EOF

echo "--- Medium prompts ---"
wrk -t"$THREADS" -c"$CONNECTIONS" -d"$DURATION" \
  --script="$LUA_SCRIPT" \
  "$API_URL/v1/chat/completions"

rm -f "$LUA_SCRIPT"
echo ""
echo "Benchmark complete."
