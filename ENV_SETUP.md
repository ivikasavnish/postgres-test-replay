# Environment Variables Configuration

The postgres-test-replay system now supports configuration via `.env` files for easier deployment and management.

## Quick Start

1. Copy the example environment file:
```bash
cp .env.example .env
```

2. Edit `.env` with your database connection strings:
```bash
INPUT_DSN=postgres://postgres:postgres@localhost:5432/testdb?sslmode=disable
OUTPUT_DSN=postgres://postgres:postgres@localhost:5433/testdb?sslmode=disable
```

3. Start the application (it will automatically load `.env`):
```bash
./postgres-test-replay -mode ipc
```

## Environment Variables

### Required

- **INPUT_DSN**: PostgreSQL connection string for the primary (input) database
  - Format: `postgres://user:password@host:port/database?sslmode=disable`
  - Example: `postgres://postgres:postgres@localhost:5432/testdb?sslmode=disable`

- **OUTPUT_DSN**: PostgreSQL connection string for the replica (output) database
  - Format: `postgres://user:password@host:port/database?sslmode=disable`
  - Example: `postgres://postgres:postgres@localhost:5433/testdb?sslmode=disable`

### Optional

- **SERVER_PORT**: Port for the IPC server (default: 8080)
- **SERVER_UI_PATH**: Path to UI files (default: ./ui)
- **WAL_LOG_PATH**: Directory for WAL log files (default: ./waldata)
- **BACKUP_PATH**: Directory for backup files (default: ./backups)
- **SESSION_PATH**: Directory for session data (default: ./sessions)
- **CHECKPOINT_PATH**: Directory for checkpoint data (default: ./checkpoints)
- **REPLICATION_SLOT**: Name of the replication slot (default: test_slot)
- **PUBLICATION_NAME**: Name of the publication (default: test_publication)

## Web UI

The system now includes a minimal web UI for easy management:

### Accessing the UI

1. Start the IPC server:
```bash
./postgres-test-replay -mode ipc
```

2. Open your browser to:
```
http://localhost:8080
```

### UI Features

1. **Database DSN Display**
   - Shows connection strings for both input and output databases
   - Visual status indicators for connection health

2. **Fixed Checkpoints**
   - **Database Creation**: Represents the initial state when the database was created
   - **Initial Migration**: Represents the state after the first schema migration
   - These are permanent reference points you can always return to

3. **Checkpoint Management**
   - Create new checkpoints with custom names and descriptions
   - View all checkpoints with timestamps
   - Click on any checkpoint to navigate to that point in time

4. **WAL Log Viewer**
   - Real-time view of database operations (INSERT, UPDATE, DELETE, DDL)
   - Color-coded operations for easy identification
   - Scroll through log entries
   - Auto-refresh every 5 seconds

5. **Statistics Dashboard**
   - Total WAL log entries
   - Total checkpoints

### UI Actions

**Scroll Log**: Use the scroll buttons to navigate through WAL entries
- ⬆️ Scroll to Top
- ⬇️ Scroll to Bottom

**Apply Checkpoint**: Click on any checkpoint in the list to load its state

**Run to Checkpoint**: Navigate to a specific checkpoint to view all entries up to that point

**Create Checkpoint**: Click "➕ Create Checkpoint" to create a new checkpoint at the current position

## Command Line Usage

### Using .env file (recommended)
```bash
# Will automatically load .env from current directory
./postgres-test-replay -mode ipc

# Specify custom .env location
./postgres-test-replay -env /path/to/.env -mode ipc
```

### Using JSON config (legacy)
```bash
./postgres-test-replay -config config.json -mode ipc
```

### Override server address
```bash
./postgres-test-replay -mode ipc -addr :9090
```

## Configuration Priority

The system loads configuration in the following order (later overrides earlier):

1. Default values
2. .env file (if specified with `-env` flag or `.env` exists)
3. JSON config file (if specified with `-config` flag)
4. Command-line flags (e.g., `-addr`)

## Fixed Checkpoints

The system provides two special fixed checkpoints:

1. **CREATION**: Represents the database state at creation time
   - This is the absolute starting point
   - Useful for complete resets

2. **MIGRATION**: Represents the state after initial schema migration
   - Includes the base schema and initial data
   - Useful as a clean starting point for testing

These checkpoints are always available and cannot be deleted. They serve as stable reference points for testing and debugging.

## Example: Complete Workflow

```bash
# 1. Setup environment
cp .env.example .env
# Edit .env with your database connections

# 2. Start Docker databases
docker-compose up -d

# 3. Start the WAL listener (captures changes)
./postgres-test-replay -mode listener &

# 4. Start the IPC server with UI
./postgres-test-replay -mode ipc

# 5. Open browser to http://localhost:8080

# 6. Make database changes
psql -h localhost -p 5432 -U postgres -d testdb -c "INSERT INTO test_table (name, value) VALUES ('test', 100)"

# 7. Create checkpoint via UI or API
curl -X POST http://localhost:8080/api/checkpoints \
  -H "Content-Type: application/json" \
  -d '{"name":"After insert","description":"State after test insert","lsn":"0/0","entry_index":0,"session_id":"..."}'

# 8. View changes in UI - they will auto-refresh
# 9. Click on checkpoint to navigate to that state
# 10. Use fixed checkpoints to reset to known states
```

## Troubleshooting

### UI not loading
- Check that SERVER_UI_PATH points to the correct directory
- Verify `ui/index.html` exists
- Check browser console for errors

### DSN not showing
- Verify .env file is in the correct location
- Check that INPUT_DSN and OUTPUT_DSN are properly formatted
- Look at server logs for configuration errors

### Checkpoints not appearing
- Ensure checkpoint data directory exists and is writable
- Check that sessions have been created
- Verify API is responding: `curl http://localhost:8080/health`

## Security Notes

- The .env file contains sensitive credentials - never commit it to version control
- Use `.env.example` as a template without real credentials
- Restrict access to the web UI in production environments
- Consider using environment variables from your deployment platform instead of .env files in production
