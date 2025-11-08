#!/bin/bash

# Health check endpoint
# Returns the health status of the API server

API_URL="${API_URL:-http://localhost:8000}"

curl -X GET "${API_URL}/health" \
  -H "Content-Type: application/json" \
  -w "\nHTTP Status: %{http_code}\n"

