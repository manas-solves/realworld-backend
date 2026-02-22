#!/bin/bash

# Load test script for article creation using hey
# This script registers/logs in a user and runs hey load test
# to create articles for a specified duration and rate
# 
# Environment variables:
#   API_URL - API server URL (default: http://localhost:4000)
#   LOADTEST_DURATION - Duration in seconds (default: 30)
#   LOADTEST_RATE - Requests per second (default: 10)
#   NUM_WORKERS - Number of workers (default: 2)
set -e

# Use environment variables for configuration
API_URL="${API_URL:-http://localhost:4000}"
LOADTEST_DURATION="${LOADTEST_DURATION:-30}"  # Duration in seconds
LOADTEST_RATE="${LOADTEST_RATE:-10}"  # Requests per second
NUM_WORKERS="${NUM_WORKERS:-2}"  # Number of workers
LOADTEST_USERNAME="bob"
LOADTEST_EMAIL="bob@example.com"
LOADTEST_PASSWORD="password123"

echo "Starting load test setup..."
echo "API URL: ${API_URL}"

# Check if hey is available
if ! command -v hey &> /dev/null; then
  echo "Error: hey not found in PATH. Please install it:"
  echo "  go install github.com/rakyll/hey@latest"
  exit 1
fi

# Register a dummy user called bob
echo "Registering user 'bob'..."
REGISTER_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/users" \
  -H "Content-Type: application/json" \
  -d "{
    \"user\": {
      \"username\": \"${LOADTEST_USERNAME}\",
      \"email\": \"${LOADTEST_EMAIL}\",
      \"password\": \"${LOADTEST_PASSWORD}\"
    }
  }" 2>/dev/null || echo -e "\n000")

# Extract HTTP status code and response body
HTTP_CODE=$(echo "$REGISTER_RESPONSE" | tail -1 || echo "000")
RESPONSE_BODY=$(echo "$REGISTER_RESPONSE" | sed '$d' || echo "")

# Try to extract token from registration response
TOKEN=$(echo "$RESPONSE_BODY" | grep -o '"token":"[^"]*' | cut -d'"' -f4 || echo "")

# Login the dummy user called bob and get the token
if [ -z "$TOKEN" ]; then
  echo "User already exists, logging in..."
  LOGIN_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/users/login" \
    -H "Content-Type: application/json" \
    -d "{
      \"user\": {
        \"email\": \"${LOADTEST_EMAIL}\",
        \"password\": \"${LOADTEST_PASSWORD}\"
      }
    }" 2>/dev/null || echo -e "\n000")
  LOGIN_HTTP_CODE=$(echo "$LOGIN_RESPONSE" | tail -1 || echo "000")
  LOGIN_BODY=$(echo "$LOGIN_RESPONSE" | sed '$d' || echo "")
  TOKEN=$(echo "$LOGIN_BODY" | grep -o '"token":"[^"]*' | cut -d'"' -f4 || echo "")
fi

if [ -z "$TOKEN" ]; then
  echo "Error: Failed to get authentication token"
  echo "Register HTTP code: ${HTTP_CODE:-unknown}"
  echo "Register response: ${RESPONSE_BODY:-none}"
  if [ -n "$LOGIN_HTTP_CODE" ]; then
    echo "Login HTTP code: ${LOGIN_HTTP_CODE}"
    echo "Login response: ${LOGIN_BODY:-none}"
  fi
  echo ""
  echo "Make sure the API server is running at ${API_URL}"
  exit 1
fi

echo "Authentication successful. Token obtained."
echo "Starting load test: ${NUM_WORKERS} workers, ${LOADTEST_RATE} requests/second per worker for ${LOADTEST_DURATION} seconds..."
echo ""

# Use bob's authentication token to create articles
# Use hey to run the load test - https://github.com/rakyll/hey
# -n: total number of requests
# -z: duration of the test in seconds
# -q: rate limit (requests per second)
# -c: number of workers (concurrency)
# -m: HTTP method
# -H: headers
# -d: request body (same for all requests)

hey -z ${LOADTEST_DURATION}s \
    -q ${LOADTEST_RATE} \
    -c ${NUM_WORKERS} \
    -m POST \
    -H "Authorization: Token ${TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{"article":{"title":"Load Test Article","description":"This is a load test article","body":"This is the body of the load test article","tagList":["loadtest","performance"]}}' \
    "${API_URL}/articles"

echo ""
echo "Load test completed!"
