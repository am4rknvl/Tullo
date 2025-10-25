# Getting Started with Tullo

This guide will help you set up and run the Tullo messaging platform locally.

## Prerequisites

- **Go 1.21+** - [Download](https://golang.org/dl/)
- **PostgreSQL 14+** - [Download](https://www.postgresql.org/download/)
- **Redis 7+** - [Download](https://redis.io/download)
- **Node.js 18+** - [Download](https://nodejs.org/) (for SDK and examples)
- **Docker & Docker Compose** (optional but recommended) - [Download](https://www.docker.com/products/docker-desktop)

## Quick Start with Docker (Recommended)

The easiest way to get started is using Docker Compose:

```bash
# 1. Clone the repository
git clone https://github.com/am4rknvl/Tullo.git
cd Tullo

# 2. Start all services (Postgres, Redis, Backend)
docker-compose up -d

# 3. Check logs
docker-compose logs -f backend

# 4. Stop services
docker-compose down
```

The backend will be available at `http://localhost:8080`

## Manual Setup

If you prefer to run services manually:

### 1. Set Up Environment Variables

```bash
cp .env.example .env
```

Edit `.env` and configure your database and Redis settings.

### 2. Install Go Dependencies

```bash
go mod download
```

### 3. Start PostgreSQL and Redis

Make sure PostgreSQL and Redis are running on your system.

**PostgreSQL:**
```bash
# Create database
createdb tullo_db

# Or using psql
psql -U postgres
CREATE DATABASE tullo_db;
```

**Redis:**
```bash
redis-server
```

### 4. Run Database Migrations

```bash
go run cmd/migrate/main.go up
```

### 5. Start the Backend Server

```bash
go run cmd/server/main.go
```

The server will start on `http://localhost:8080`

## Testing the API

### Health Check

```bash
curl http://localhost:8080/health
```

### Register a User

```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123",
    "display_name": "John Doe"
  }'
```

Response:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "uuid-here",
    "email": "user@example.com",
    "display_name": "John Doe",
    "created_at": "2025-10-25T12:00:00Z",
    "updated_at": "2025-10-25T12:00:00Z"
  }
}
```

### Login

```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'
```

### Create a Conversation

```bash
TOKEN="your-jwt-token-here"

curl -X POST http://localhost:8080/api/v1/conversations \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "is_group": false,
    "members": ["other-user-id"]
  }'
```

### Send a Message

```bash
curl -X POST http://localhost:8080/api/v1/messages \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "conversation_id": "conversation-id",
    "body": "Hello, World!"
  }'
```

### Get Messages

```bash
curl "http://localhost:8080/api/v1/messages?conversation_id=conversation-id&limit=50" \
  -H "Authorization: Bearer $TOKEN"
```

## WebSocket Connection

Connect to WebSocket for real-time messaging:

```javascript
const ws = new WebSocket('ws://localhost:8080/ws?token=YOUR_JWT_TOKEN');

ws.onopen = () => {
  console.log('Connected!');
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log('Received:', message);
};

// Send a message
ws.send(JSON.stringify({
  event: 'message.send',
  payload: {
    conversation_id: 'conv-id',
    body: 'Hello via WebSocket!'
  }
}));
```

## Running the JavaScript SDK Example

### 1. Build the SDK

```bash
cd sdk/javascript
npm install
npm run build
```

### 2. Run the React Example

```bash
cd ../../examples/react-chat
npm install
npm run dev
```

Open `http://localhost:3000` in your browser.

## Project Structure

```
Tullo/
├── cmd/
│   ├── server/          # Main server entry point
│   └── migrate/         # Database migration tool
├── config/              # Configuration management
├── internal/
│   ├── auth/           # JWT and password handling
│   ├── cache/          # Redis client
│   ├── database/       # Postgres connection and migrations
│   ├── handlers/       # HTTP request handlers
│   ├── middleware/     # Auth, CORS, rate limiting
│   ├── models/         # Data models
│   ├── repository/     # Database repositories
│   └── websocket/      # WebSocket hub and client
├── sdk/
│   └── javascript/     # JavaScript/TypeScript SDK
├── examples/
│   └── react-chat/     # React chat example
├── docker-compose.yml  # Docker services
├── Dockerfile          # Backend container
└── README.md
```

## Development Workflow

### 1. Make Changes

Edit Go files in the project.

### 2. Run Tests

```bash
go test ./...
```

### 3. Restart Server

```bash
# If running manually
go run cmd/server/main.go

# If using Docker
docker-compose restart backend
```

### 4. Create New Migration

```bash
# Add your migration to internal/database/migrations.go
# Then run:
go run cmd/migrate/main.go up
```

## Environment Variables

Key environment variables (see `.env.example` for all):

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `DB_HOST` | PostgreSQL host | `localhost` |
| `DB_PORT` | PostgreSQL port | `5432` |
| `DB_NAME` | Database name | `tullo_db` |
| `REDIS_HOST` | Redis host | `localhost` |
| `REDIS_PORT` | Redis port | `6379` |
| `JWT_SECRET` | JWT signing secret | (required in production) |
| `CORS_ALLOWED_ORIGINS` | Allowed CORS origins | `http://localhost:3000` |

## Troubleshooting

### Database Connection Error

```
Failed to connect to database: connection refused
```

**Solution:** Make sure PostgreSQL is running and credentials in `.env` are correct.

### Redis Connection Error

```
Failed to connect to Redis: connection refused
```

**Solution:** Make sure Redis is running on the configured port.

### Port Already in Use

```
bind: address already in use
```

**Solution:** Change the `PORT` in `.env` or stop the process using port 8080.

### Migration Errors

```
Migration failed: relation already exists
```

**Solution:** Check migration status:
```bash
go run cmd/migrate/main.go status
```

## Next Steps

- Read the [API Documentation](README.md#api-documentation)
- Explore the [JavaScript SDK](sdk/javascript/README.md)
- Try the [React Example](examples/react-chat/README.md)
- Check out the [WebSocket Events](README.md#websocket-events)

## Support

- **Issues:** [GitHub Issues](https://github.com/am4rknvl/Tullo/issues)
- **Discussions:** [GitHub Discussions](https://github.com/am4rknvl/Tullo/discussions)

## License

MIT License - see [LICENSE](LICENSE) file for details.
