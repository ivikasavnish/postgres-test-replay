# postgres-test-replay

A comprehensive Golang-based solution for PostgreSQL database testing, replay, and time-travel debugging. This tool provides the ability to capture database changes via logical replication, create checkpoints, and replay database states.

## Features

1. **PostgreSQL Backup Creation** - Create and manage database backups
2. **Docker Compose with Logical Replication** - Pre-configured setup for primary and replica databases
3. **WAL Replication Listener** - Capture INSERT, UPDATE, DELETE, and DDL operations in real-time
4. **Checkpoint Management** - Create markers at specific points in the transaction log
5. **IPC Service** - REST API to navigate through checkpoints and apply transactions
6. **Session Management** - Create, switch, and manage multiple replay sessions
7. **Database Replay** - Replay transactions from any checkpoint
8. **Database Differential Support** - Track changes per database
9. **🆕 Web UI** - Minimal web interface for visual management and monitoring
10. **🆕 Environment Configuration** - Simple .env file-based configuration with DSN support
11. **🆕 Fixed Checkpoints** - Two permanent reference points (Creation and Migration)

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│  Primary DB │────▶│  Replication │────▶│  WAL Logs   │
│  (Postgres) │     │   Listener   │     │   (Files)   │
└─────────────┘     └──────────────┘     └─────────────┘
                                                 │
                                                 ▼
                    ┌──────────────┐     ┌─────────────┐
                    │ IPC Service  │────▶│ Checkpoints │
                    │  (REST API)  │     │   Manager   │
                    └──────────────┘     └─────────────┘
                           │
                           ▼
                    ┌──────────────┐     ┌─────────────┐
                    │   Session    │────▶│  Replica DB │
                    │   Manager    │     │  (Postgres) │
                    └──────────────┘     └─────────────┘
```

## Prerequisites

- Go 1.19 or higher
- Docker and Docker Compose
- PostgreSQL client tools (pg_dump, pg_restore)

## Installation

```bash
# Clone the repository
git clone https://github.com/ivikasavnish/postgres-test-replay.git
cd postgres-test-replay

# Install dependencies
go mod download

# Build the application
go build -o postgres-test-replay ./cmd/postgres-test-replay
```

## Quick Start

### 1. Start PostgreSQL with Docker Compose

```bash
docker-compose up -d
```

This starts two PostgreSQL instances:
- Primary: `localhost:5432`
- Replica: `localhost:5433`

### 2. Setup Configuration (Choose One)

**Option A: Using .env file (Recommended)**

```bash
# Copy example environment file
cp .env.example .env

# Edit .env with your database connections
# INPUT_DSN=postgres://postgres:postgres@localhost:5432/testdb?sslmode=disable
# OUTPUT_DSN=postgres://postgres:postgres@localhost:5433/testdb?sslmode=disable
```

**Option B: Using JSON config**

```bash
cp config.example.json config.json
# Edit config.json with your database settings
```

### 3. Start the Services

**Terminal 1 - Start WAL Listener:**
```bash
./postgres-test-replay -mode listener
```

**Terminal 2 - Start IPC Server with Web UI:**
```bash
./postgres-test-replay -mode ipc
```

**Terminal 3 - Open Web UI:**
```bash
# Open http://localhost:8080 in your browser
```

### 4. Use the Web UI

The web interface provides:
- 📊 Real-time database connection status with DSN display
- 📌 Fixed checkpoints (Database Creation, Initial Migration)
- ➕ Create custom checkpoints
- 📝 View and scroll through WAL logs
- 🎯 Navigate to any checkpoint
- 📈 Statistics dashboard

### 5. Create a Database Backup (Optional)

```bash
./postgres-test-replay -mode backup
```

### 4. Start the WAL Replication Listener

```bash
./postgres-test-replay -mode listener
```

This will start capturing all database changes (INSERT, UPDATE, DELETE, DDL) to WAL log files.

### 5. Start the IPC Server

In a new terminal:

```bash
./postgres-test-replay -mode ipc -addr :8080
```

## API Usage

### Create a Session

```bash
curl -X POST http://localhost:8080/api/sessions \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Test Session",
    "description": "Testing session for feature X",
    "database": "testdb"
  }'
```

### Create a Checkpoint

```bash
curl -X POST http://localhost:8080/api/checkpoints \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Before migration",
    "description": "State before running migration",
    "lsn": "0/1234567",
    "entry_index": 42,
    "session_id": "session-uuid"
  }'
```

### List Sessions

```bash
curl http://localhost:8080/api/sessions
```

### Switch to a Session

```bash
curl -X POST http://localhost:8080/api/sessions/switch \
  -H "Content-Type: application/json" \
  -d '{"session_id": "session-uuid"}'
```

### Navigate to a Checkpoint

```bash
curl -X POST http://localhost:8080/api/navigate \
  -H "Content-Type: application/json" \
  -d '{"checkpoint_id": "checkpoint-uuid"}'
```

### Replay a Session

```bash
curl -X POST http://localhost:8080/api/replay \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "session-uuid",
    "checkpoint_id": "checkpoint-uuid"
  }'
```

### List Checkpoints

```bash
# All checkpoints
curl http://localhost:8080/api/checkpoints

# Checkpoints for a specific session
curl "http://localhost:8080/api/checkpoints?session_id=session-uuid"
```

## Configuration

The configuration file (`config.json`) contains settings for:

- **primary_db**: Primary database connection settings
- **replica_db**: Replica database connection settings
- **storage**: Paths for WAL logs, backups, sessions, and checkpoints
- **replication**: Replication slot and publication names

See `config.example.json` for a complete example.

## Use Cases

### 1. Testing Database Migrations

```bash
# Create a checkpoint before migration
curl -X POST http://localhost:8080/api/checkpoints \
  -d '{"name": "pre-migration", ...}'

# Run migration
psql -h localhost -U postgres -d testdb -f migration.sql

# Create a checkpoint after migration
curl -X POST http://localhost:8080/api/checkpoints \
  -d '{"name": "post-migration", ...}'

# Replay to test rollback scenarios
curl -X POST http://localhost:8080/api/replay \
  -d '{"checkpoint_id": "pre-migration-uuid"}'
```

### 2. Time-Travel Debugging

```bash
# Create checkpoints at different points
# Navigate between them to debug issues
curl -X POST http://localhost:8080/api/navigate \
  -d '{"checkpoint_id": "specific-point-uuid"}'
```

### 3. Multiple Test Sessions

```bash
# Create different sessions for different test scenarios
# Session 1: Performance testing
# Session 2: Edge case testing
# Session 3: Integration testing

# Switch between them as needed
curl -X POST http://localhost:8080/api/sessions/switch \
  -d '{"session_id": "performance-test-uuid"}'
```

## Project Structure

```
.
├── cmd/
│   └── postgres-test-replay/    # Main application
│       └── main.go
├── pkg/
│   ├── backup/                  # Backup management
│   ├── checkpoint/              # Checkpoint system
│   ├── config/                  # Configuration handling
│   ├── ipc/                     # IPC REST API server
│   ├── replication/             # WAL replication listener
│   ├── session/                 # Session management
│   └── wal/                     # WAL log handling
├── docker-compose.yml           # PostgreSQL setup
├── init-primary.sql             # Database initialization
└── config.example.json          # Example configuration
```

## Development

### Build

```bash
go build -o postgres-test-replay ./cmd/postgres-test-replay
```

### Run Tests

```bash
go test ./...
```

### Format Code

```bash
go fmt ./...
```

## Troubleshooting

### Replication Slot Already Exists

If you see an error about the replication slot already existing:

```sql
-- Connect to the primary database
psql -h localhost -U postgres -d testdb

-- Drop the existing slot
SELECT pg_drop_replication_slot('test_slot');
```

### Permission Denied

Ensure the application has write permissions to the storage directories:

```bash
mkdir -p waldata backups sessions checkpoints
chmod 755 waldata backups sessions checkpoints
```

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
