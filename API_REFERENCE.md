# Tullo API Reference

Complete API documentation for the Tullo messaging platform.

## Base URL

```
http://localhost:8080
```

## Authentication

All protected endpoints require a JWT token in the Authorization header:

```
Authorization: Bearer <token>
```

Get a token by registering or logging in.

---

## Authentication Endpoints

### Register User

Create a new user account.

**Endpoint:** `POST /auth/register`

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "password123",
  "display_name": "John Doe",
  "avatar_url": "https://example.com/avatar.jpg" // optional
}
```

**Response:** `201 Created`
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "display_name": "John Doe",
    "avatar_url": "https://example.com/avatar.jpg",
    "created_at": "2025-10-25T12:00:00Z",
    "updated_at": "2025-10-25T12:00:00Z"
  }
}
```

**Errors:**
- `400 Bad Request` - Invalid request body
- `500 Internal Server Error` - Server error

---

### Login

Authenticate an existing user.

**Endpoint:** `POST /auth/login`

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**Response:** `200 OK`
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "display_name": "John Doe",
    "avatar_url": "https://example.com/avatar.jpg",
    "created_at": "2025-10-25T12:00:00Z",
    "updated_at": "2025-10-25T12:00:00Z"
  }
}
```

**Errors:**
- `400 Bad Request` - Invalid request body
- `401 Unauthorized` - Invalid credentials

---

### Get Current User

Get the authenticated user's information.

**Endpoint:** `GET /api/v1/me`

**Headers:**
```
Authorization: Bearer <token>
```

**Response:** `200 OK`
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "display_name": "John Doe",
  "avatar_url": "https://example.com/avatar.jpg",
  "created_at": "2025-10-25T12:00:00Z",
  "updated_at": "2025-10-25T12:00:00Z"
}
```

**Errors:**
- `401 Unauthorized` - Invalid or missing token
- `404 Not Found` - User not found

---

## Conversation Endpoints

### List Conversations

Get all conversations for the authenticated user.

**Endpoint:** `GET /api/v1/conversations`

**Headers:**
```
Authorization: Bearer <token>
```

**Response:** `200 OK`
```json
[
  {
    "id": "conv-id-1",
    "is_group": false,
    "name": null,
    "created_at": "2025-10-25T12:00:00Z",
    "updated_at": "2025-10-25T12:00:00Z",
    "members": [
      {
        "id": "user-id-1",
        "email": "user1@example.com",
        "display_name": "User One",
        "avatar_url": null,
        "created_at": "2025-10-25T12:00:00Z",
        "updated_at": "2025-10-25T12:00:00Z"
      }
    ],
    "last_message": {
      "id": "msg-id",
      "conversation_id": "conv-id-1",
      "sender_id": "user-id-1",
      "body": "Hello!",
      "created_at": "2025-10-25T12:05:00Z",
      "updated_at": "2025-10-25T12:05:00Z"
    }
  }
]
```

---

### Get Conversation

Get a specific conversation by ID.

**Endpoint:** `GET /api/v1/conversations/:id`

**Headers:**
```
Authorization: Bearer <token>
```

**Response:** `200 OK`
```json
{
  "id": "conv-id-1",
  "is_group": true,
  "name": "Team Chat",
  "created_at": "2025-10-25T12:00:00Z",
  "updated_at": "2025-10-25T12:00:00Z",
  "members": [
    {
      "id": "user-id-1",
      "email": "user1@example.com",
      "display_name": "User One",
      "avatar_url": null,
      "created_at": "2025-10-25T12:00:00Z",
      "updated_at": "2025-10-25T12:00:00Z"
    }
  ]
}
```

**Errors:**
- `400 Bad Request` - Invalid conversation ID
- `403 Forbidden` - Not a member of the conversation
- `404 Not Found` - Conversation not found

---

### Create Conversation

Create a new 1:1 or group conversation.

**Endpoint:** `POST /api/v1/conversations`

**Headers:**
```
Authorization: Bearer <token>
```

**Request Body (1:1):**
```json
{
  "is_group": false,
  "members": ["user-id-2"]
}
```

**Request Body (Group):**
```json
{
  "is_group": true,
  "name": "Team Chat",
  "members": ["user-id-2", "user-id-3"]
}
```

**Response:** `201 Created`
```json
{
  "id": "new-conv-id",
  "is_group": true,
  "name": "Team Chat",
  "created_at": "2025-10-25T12:00:00Z",
  "updated_at": "2025-10-25T12:00:00Z",
  "members": [...]
}
```

**Errors:**
- `400 Bad Request` - Invalid request body
- `500 Internal Server Error` - Failed to create conversation

---

### Add Members

Add members to a group conversation.

**Endpoint:** `POST /api/v1/conversations/:id/members`

**Headers:**
```
Authorization: Bearer <token>
```

**Request Body:**
```json
{
  "members": ["user-id-4", "user-id-5"]
}
```

**Response:** `200 OK`
```json
{
  "message": "Members added successfully"
}
```

**Errors:**
- `400 Bad Request` - Cannot add members to 1:1 conversation
- `403 Forbidden` - Not a member of the conversation
- `404 Not Found` - Conversation not found

---

### Remove Member

Remove a member from a conversation.

**Endpoint:** `DELETE /api/v1/conversations/:id/members/:user_id`

**Headers:**
```
Authorization: Bearer <token>
```

**Response:** `200 OK`
```json
{
  "message": "Member removed successfully"
}
```

**Errors:**
- `403 Forbidden` - Not authorized
- `404 Not Found` - Member not found
- `500 Internal Server Error` - Failed to remove member

---

## Message Endpoints

### Get Messages

Get messages for a conversation with pagination.

**Endpoint:** `GET /api/v1/messages`

**Headers:**
```
Authorization: Bearer <token>
```

**Query Parameters:**
- `conversation_id` (required) - Conversation ID
- `limit` (optional) - Number of messages (default: 50, max: 100)
- `offset` (optional) - Offset for pagination (default: 0)

**Example:**
```
GET /api/v1/messages?conversation_id=conv-id&limit=50&offset=0
```

**Response:** `200 OK`
```json
[
  {
    "id": "msg-id-1",
    "conversation_id": "conv-id",
    "sender_id": "user-id-1",
    "body": "Hello, World!",
    "created_at": "2025-10-25T12:00:00Z",
    "updated_at": "2025-10-25T12:00:00Z",
    "sender": {
      "id": "user-id-1",
      "email": "user1@example.com",
      "display_name": "User One",
      "avatar_url": null,
      "created_at": "2025-10-25T11:00:00Z",
      "updated_at": "2025-10-25T11:00:00Z"
    }
  }
]
```

**Errors:**
- `400 Bad Request` - Missing conversation_id
- `403 Forbidden` - Not a member of the conversation
- `500 Internal Server Error` - Failed to get messages

---

### Send Message

Send a message to a conversation via REST API.

**Endpoint:** `POST /api/v1/messages`

**Headers:**
```
Authorization: Bearer <token>
```

**Request Body:**
```json
{
  "conversation_id": "conv-id",
  "body": "Hello, World!"
}
```

**Response:** `201 Created`
```json
{
  "id": "new-msg-id",
  "conversation_id": "conv-id",
  "sender_id": "user-id",
  "body": "Hello, World!",
  "created_at": "2025-10-25T12:00:00Z",
  "updated_at": "2025-10-25T12:00:00Z"
}
```

**Rate Limiting:** 10 messages per second per user

**Errors:**
- `400 Bad Request` - Invalid request body
- `403 Forbidden` - Not a member of the conversation
- `429 Too Many Requests` - Rate limit exceeded
- `500 Internal Server Error` - Failed to send message

---

### Mark Message as Read

Mark a message as read.

**Endpoint:** `PUT /api/v1/messages/:id/read`

**Headers:**
```
Authorization: Bearer <token>
```

**Response:** `200 OK`
```json
{
  "message": "Message marked as read"
}
```

**Errors:**
- `400 Bad Request` - Invalid message ID
- `403 Forbidden` - Not authorized
- `404 Not Found` - Message not found

---

## WebSocket API

### Connection

Connect to the WebSocket server for real-time messaging.

**Endpoint:** `ws://localhost:8080/ws?token=<JWT_TOKEN>`

**Connection:**
```javascript
const ws = new WebSocket('ws://localhost:8080/ws?token=YOUR_JWT_TOKEN');
```

---

### Client → Server Events

#### Send Message

```json
{
  "event": "message.send",
  "payload": {
    "conversation_id": "conv-id",
    "body": "Hello via WebSocket!"
  }
}
```

#### Mark Message as Read

```json
{
  "event": "message.read",
  "payload": {
    "message_id": "msg-id",
    "conversation_id": "conv-id"
  }
}
```

#### Start Typing

```json
{
  "event": "typing.start",
  "payload": {
    "conversation_id": "conv-id"
  }
}
```

#### Stop Typing

```json
{
  "event": "typing.stop",
  "payload": {
    "conversation_id": "conv-id"
  }
}
```

---

### Server → Client Events

#### New Message

```json
{
  "event": "message.new",
  "payload": {
    "id": "msg-id",
    "conversation_id": "conv-id",
    "sender_id": "user-id",
    "body": "Hello!",
    "created_at": "2025-10-25T12:00:00Z",
    "updated_at": "2025-10-25T12:00:00Z"
  }
}
```

#### Message Read

```json
{
  "event": "message.read",
  "payload": {
    "message_id": "msg-id",
    "conversation_id": "conv-id",
    "user_id": "user-id",
    "read_at": "2025-10-25T12:01:00Z"
  }
}
```

#### Typing Start

```json
{
  "event": "typing.start",
  "payload": {
    "conversation_id": "conv-id",
    "user_id": "user-id",
    "is_typing": true
  }
}
```

#### Typing Stop

```json
{
  "event": "typing.stop",
  "payload": {
    "conversation_id": "conv-id",
    "user_id": "user-id",
    "is_typing": false
  }
}
```

#### Presence Update

```json
{
  "event": "presence.update",
  "payload": {
    "user_id": "user-id",
    "status": "online",
    "last_seen": "2025-10-25T12:00:00Z"
  }
}
```

#### Error

```json
{
  "event": "error",
  "payload": {
    "message": "Error description",
    "code": "ERROR_CODE"
  }
}
```

---

## Error Responses

All error responses follow this format:

```json
{
  "error": "Error message description"
}
```

### Common HTTP Status Codes

- `200 OK` - Request successful
- `201 Created` - Resource created successfully
- `400 Bad Request` - Invalid request parameters
- `401 Unauthorized` - Authentication required or failed
- `403 Forbidden` - Insufficient permissions
- `404 Not Found` - Resource not found
- `429 Too Many Requests` - Rate limit exceeded
- `500 Internal Server Error` - Server error

---

## Rate Limiting

- **Messages:** 10 messages per second per user
- **WebSocket:** Automatic reconnection with exponential backoff

---

## Pagination

Endpoints that return lists support pagination:

- `limit` - Number of items (default: 50, max: 100)
- `offset` - Number of items to skip (default: 0)

Example:
```
GET /api/v1/messages?conversation_id=conv-id&limit=20&offset=40
```

---

## Data Types

### User

```typescript
{
  id: string (UUID)
  email: string
  display_name: string
  avatar_url?: string
  created_at: string (ISO 8601)
  updated_at: string (ISO 8601)
}
```

### Conversation

```typescript
{
  id: string (UUID)
  is_group: boolean
  name?: string
  created_at: string (ISO 8601)
  updated_at: string (ISO 8601)
  members?: User[]
  last_message?: Message
}
```

### Message

```typescript
{
  id: string (UUID)
  conversation_id: string (UUID)
  sender_id: string (UUID)
  body: string
  created_at: string (ISO 8601)
  updated_at: string (ISO 8601)
  sender?: User
}
```

---

## Best Practices

1. **Use WebSocket for real-time messaging** - Faster than REST API
2. **Implement exponential backoff** - For reconnection attempts
3. **Cache user data** - Reduce API calls
4. **Handle errors gracefully** - Show user-friendly messages
5. **Validate input** - Before sending to API
6. **Use pagination** - For large message lists
7. **Implement typing indicators** - Improve UX
8. **Show presence status** - Keep users informed

---

## SDK Usage

For easier integration, use the official JavaScript SDK:

```bash
npm install @tullo/sdk
```

See [SDK Documentation](sdk/javascript/README.md) for details.
