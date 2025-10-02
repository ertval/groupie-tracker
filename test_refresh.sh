#!/bin/bash
# Test script for manual refresh endpoint

echo "Testing manual refresh endpoint..."
echo ""

# Test with POST (should work)
echo "1. Testing POST request to /api/refresh:"
curl -X POST http://localhost:8080/api/refresh
echo ""
echo ""

# Test with GET (should fail with 405)
echo "2. Testing GET request to /api/refresh (should fail):"
curl -X GET http://localhost:8080/api/refresh
echo ""
echo ""

echo "3. Checking health endpoint:"
curl http://localhost:8080/health
echo ""
