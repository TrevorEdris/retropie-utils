#!/bin/bash

# Get sync status
# Returns the current status of the syncer including whether a sync is running
# and information about the last sync operation

API_URL="${API_URL:-http://localhost:8000}"

response=$(curl -s -X GET "${API_URL}/status" \
  -H "Content-Type: application/json" \
  -w "\nHTTP_STATUS:%{http_code}")

http_code=$(echo "$response" | grep -o "HTTP_STATUS:[0-9]*" | cut -d: -f2)
body=$(echo "$response" | sed '/HTTP_STATUS:/d')

echo "$body" | jq '.' 2>/dev/null || echo "$body"
echo "HTTP Status: $http_code"

if [ "$http_code" -eq 200 ]; then
  is_running=$(echo "$body" | jq -r '.is_running' 2>/dev/null)
  if [ "$is_running" = "true" ]; then
    echo "Status: Sync operation is currently running"
  else
    echo "Status: No sync operation in progress"
  fi
  exit 0
else
  echo "✗ Failed to get status"
  exit 1
fi

