#!/bin/bash

# Trigger sync operation
# Starts an asynchronous sync operation and returns immediately

API_URL="${API_URL:-http://localhost:8000}"

response=$(curl -s -X POST "${API_URL}/sync" \
  -H "Content-Type: application/json" \
  -w "\nHTTP_STATUS:%{http_code}")

http_code=$(echo "$response" | grep -o "HTTP_STATUS:[0-9]*" | cut -d: -f2)
body=$(echo "$response" | sed '/HTTP_STATUS:/d')

echo "$body" | jq '.' 2>/dev/null || echo "$body"
echo "HTTP Status: $http_code"

if [ "$http_code" -eq 202 ]; then
  echo "✓ Sync operation started successfully"
  exit 0
elif [ "$http_code" -eq 409 ]; then
  echo "✗ Sync operation already in progress"
  exit 1
else
  echo "✗ Failed to start sync operation"
  exit 1
fi

