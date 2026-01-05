# API Documentation

## Base URL
```
http://localhost:8080
```

## Health Check

### GET /health

Check if the API is running.

**Response:**
```json
{
  "status": "ok"
}
```

## Sessions

### POST /api/sessions

Create a new session.

**Request Body:**
```json
{
  "name": "Session Name",
  "description": "Session description",
  "database": "testdb"
}
```

**Response:** (201 Created)
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Session Name",
  "description": "Session description",
  "created_at": "2024-01-05T10:00:00Z",
  "updated_at": "2024-01-05T10:00:00Z",
  "database": "testdb",
  "checkpoints": [],
  "active": false
}
```

### GET /api/sessions

List all sessions.

**Response:** (200 OK)
```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "Session Name",
    "description": "Session description",
    "created_at": "2024-01-05T10:00:00Z",
    "updated_at": "2024-01-05T10:00:00Z",
    "database": "testdb",
    "checkpoints": ["cp-id-1", "cp-id-2"],
    "active": true
  }
]
```

### GET /api/sessions/{id}

Get a specific session.

**Response:** (200 OK)
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Session Name",
  "description": "Session description",
  "created_at": "2024-01-05T10:00:00Z",
  "updated_at": "2024-01-05T10:00:00Z",
  "database": "testdb",
  "checkpoints": [],
  "active": false
}
```

### DELETE /api/sessions/{id}

Delete a session.

**Response:** (204 No Content)

### POST /api/sessions/switch

Switch to a different session.

**Request Body:**
```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Response:** (200 OK)

## Checkpoints

### POST /api/checkpoints

Create a checkpoint.

**Request Body:**
```json
{
  "name": "Checkpoint Name",
  "description": "Checkpoint description",
  "lsn": "0/1234567",
  "entry_index": 42,
  "session_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Response:** (201 Created)
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "name": "Checkpoint Name",
  "description": "Checkpoint description",
  "timestamp": "2024-01-05T10:00:00Z",
  "lsn": "0/1234567",
  "entry_index": 42,
  "session_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

### GET /api/checkpoints

List checkpoints.

**Query Parameters:**
- `session_id` (optional): Filter by session ID

**Response:** (200 OK)
```json
[
  {
    "id": "123e4567-e89b-12d3-a456-426614174000",
    "name": "Checkpoint Name",
    "description": "Checkpoint description",
    "timestamp": "2024-01-05T10:00:00Z",
    "lsn": "0/1234567",
    "entry_index": 42,
    "session_id": "550e8400-e29b-41d4-a716-446655440000"
  }
]
```

### GET /api/checkpoints/{id}

Get a specific checkpoint.

**Response:** (200 OK)
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "name": "Checkpoint Name",
  "description": "Checkpoint description",
  "timestamp": "2024-01-05T10:00:00Z",
  "lsn": "0/1234567",
  "entry_index": 42,
  "session_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

### DELETE /api/checkpoints/{id}

Delete a checkpoint.

**Response:** (204 No Content)

## Navigation

### POST /api/navigate

Navigate to a checkpoint or between checkpoints.

**Request Body (single checkpoint):**
```json
{
  "checkpoint_id": "123e4567-e89b-12d3-a456-426614174000"
}
```

**Request Body (between checkpoints):**
```json
{
  "start_id": "123e4567-e89b-12d3-a456-426614174000",
  "end_id": "223e4567-e89b-12d3-a456-426614174001"
}
```

**Response:** (200 OK)
```json
{
  "entries": [
    {
      "id": "entry-id-1",
      "timestamp": "2024-01-05T10:00:00Z",
      "lsn": "0/1234567",
      "operation": "INSERT",
      "schema": "public",
      "table": "test_table",
      "data": {
        "id": 1,
        "name": "test"
      }
    }
  ],
  "count": 1
}
```

## Replay

### POST /api/replay

Replay a session up to a checkpoint.

**Request Body:**
```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "checkpoint_id": "123e4567-e89b-12d3-a456-426614174000"
}
```

**Response:** (200 OK)
```json
{
  "status": "success",
  "entries_applied": 42
}
```

## Error Responses

All endpoints may return error responses with appropriate HTTP status codes:

**400 Bad Request:**
```json
{
  "error": "Invalid request body"
}
```

**404 Not Found:**
```json
{
  "error": "Session not found"
}
```

**500 Internal Server Error:**
```json
{
  "error": "Internal server error message"
}
```

## Example Usage with curl

### Complete Workflow

```bash
# 1. Check API health
curl http://localhost:8080/health

# 2. Create a session
SESSION_ID=$(curl -X POST http://localhost:8080/api/sessions \
  -H "Content-Type: application/json" \
  -d '{"name":"Test Session","description":"Testing","database":"testdb"}' \
  | jq -r '.id')

# 3. Make some database changes
psql -h localhost -U postgres -d testdb -c "INSERT INTO test_table (name, value) VALUES ('test', 100)"

# 4. Create a checkpoint
CHECKPOINT_ID=$(curl -X POST http://localhost:8080/api/checkpoints \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"After insert\",\"description\":\"State after insert\",\"lsn\":\"0/0\",\"entry_index\":0,\"session_id\":\"$SESSION_ID\"}" \
  | jq -r '.id')

# 5. Make more changes
psql -h localhost -U postgres -d testdb -c "UPDATE test_table SET value = 200 WHERE name = 'test'"

# 6. Navigate to checkpoint to see entries
curl -X POST http://localhost:8080/api/navigate \
  -H "Content-Type: application/json" \
  -d "{\"checkpoint_id\":\"$CHECKPOINT_ID\"}" | jq

# 7. Replay session to checkpoint
curl -X POST http://localhost:8080/api/replay \
  -H "Content-Type: application/json" \
  -d "{\"session_id\":\"$SESSION_ID\",\"checkpoint_id\":\"$CHECKPOINT_ID\"}" | jq
```

## Python CLI Helper

For easier interaction, use the provided Python CLI:

```bash
# Create a session
./examples/cli.py session-create "My Session" --description "Test" --database testdb

# List sessions
./examples/cli.py session-list

# Create a checkpoint
./examples/cli.py checkpoint-create "CP1" --session-id <session-id>

# List checkpoints
./examples/cli.py checkpoint-list

# Navigate to checkpoint
./examples/cli.py navigate <checkpoint-id>

# Replay session
./examples/cli.py replay <session-id> <checkpoint-id>
```
