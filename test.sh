#!/bin/bash

# Tullo API Test Script

BASE_URL="http://localhost:8080"

echo "🧪 Testing Tullo API..."
echo ""

# Health check
echo "1️⃣ Health Check"
curl -s $BASE_URL/health | jq .
echo ""

# Register user
echo "2️⃣ Registering user..."
REGISTER_RESPONSE=$(curl -s -X POST $BASE_URL/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123",
    "display_name": "Test User"
  }')

echo $REGISTER_RESPONSE | jq .
TOKEN=$(echo $REGISTER_RESPONSE | jq -r '.token')
echo "Token: $TOKEN"
echo ""

# Get current user
echo "3️⃣ Getting current user..."
curl -s $BASE_URL/api/v1/me \
  -H "Authorization: Bearer $TOKEN" | jq .
echo ""

# Get conversations
echo "4️⃣ Getting conversations..."
curl -s $BASE_URL/api/v1/conversations \
  -H "Authorization: Bearer $TOKEN" | jq .
echo ""

echo "✅ Tests completed!"
