# Usage Guide

This guide provides step-by-step instructions for using postgres-test-replay.

## Quick Start

### Step 1: Setup Environment

```bash
# Clone and build
git clone https://github.com/ivikasavnish/postgres-test-replay.git
cd postgres-test-replay
make build

# Start PostgreSQL
make docker-up

# Wait for databases to be ready (about 10 seconds)
```

### Step 2: Run the Setup Script

```bash
chmod +x examples/setup.sh
./examples/setup.sh
```

This will:
- Create configuration file
- Start PostgreSQL containers
- Insert test data
- Create initial backup

### Step 3: Start Services

In separate terminals:

**Terminal 1 - WAL Listener:**
```bash
./postgres-test-replay -mode listener
```

**Terminal 2 - IPC Server:**
```bash
./postgres-test-replay -mode ipc -addr :8080
```

### Step 4: Interact with the System

Using the Python CLI helper:

```bash
# Create a session
./examples/cli.py session-create "Test Session" --description "My first session"

# List sessions to get the session ID
./examples/cli.py session-list

# Create a checkpoint (use session ID from above)
./examples/cli.py checkpoint-create "Initial State" --session-id <SESSION_ID>

# Check API health
./examples/cli.py health
```

## Complete Workflow Example

### Scenario: Testing a Database Migration

1. **Create a session for the migration test:**

```bash
SESSION_ID=$(curl -s -X POST http://localhost:8080/api/sessions \
  -H "Content-Type: application/json" \
  -d '{"name":"Migration Test","description":"Testing schema migration","database":"testdb"}' \
  | jq -r '.id')

echo "Session ID: $SESSION_ID"
```

2. **Create a checkpoint before migration:**

```bash
CHECKPOINT_BEFORE=$(curl -s -X POST http://localhost:8080/api/checkpoints \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"Pre-migration\",\"description\":\"State before migration\",\"lsn\":\"0/0\",\"entry_index\":0,\"session_id\":\"$SESSION_ID\"}" \
  | jq -r '.id')

echo "Pre-migration Checkpoint: $CHECKPOINT_BEFORE"
```

3. **Run the migration:**

```bash
docker exec postgres-primary psql -U postgres -d testdb << EOF
ALTER TABLE test_table ADD COLUMN email VARCHAR(255);
UPDATE test_table SET email = name || '@example.com';
EOF
```

4. **Create a checkpoint after migration:**

```bash
CHECKPOINT_AFTER=$(curl -s -X POST http://localhost:8080/api/checkpoints \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"Post-migration\",\"description\":\"State after migration\",\"lsn\":\"0/0\",\"entry_index\":100,\"session_id\":\"$SESSION_ID\"}" \
  | jq -r '.id')

echo "Post-migration Checkpoint: $CHECKPOINT_AFTER"
```

5. **View changes between checkpoints:**

```bash
curl -s -X POST http://localhost:8080/api/navigate \
  -H "Content-Type: application/json" \
  -d "{\"start_id\":\"$CHECKPOINT_BEFORE\",\"end_id\":\"$CHECKPOINT_AFTER\"}" \
  | jq '.entries | length'
```

6. **Replay to pre-migration state (if needed):**

```bash
curl -s -X POST http://localhost:8080/api/replay \
  -H "Content-Type: application/json" \
  -d "{\"session_id\":\"$SESSION_ID\",\"checkpoint_id\":\"$CHECKPOINT_BEFORE\"}" \
  | jq
```

## Use Cases

### 1. Debugging Production Issues

**Problem:** Need to reproduce a production bug with exact data state.

```bash
# Create session for debugging
./examples/cli.py session-create "Debug Session" --description "Reproduce issue #123"

# Create checkpoints at different times
# Navigate between them to find when bug appeared
./examples/cli.py navigate <CHECKPOINT_ID>
```

### 2. Testing Schema Changes

**Problem:** Need to test migration scripts safely.

```bash
# 1. Create pre-migration checkpoint
# 2. Run migration
# 3. Create post-migration checkpoint
# 4. Test application
# 5. If issues, replay to pre-migration state
# 6. Fix migration script and retry
```

### 3. Performance Testing

**Problem:** Need to test performance with specific data states.

```bash
# Create checkpoints at different data volumes
# Replay to each state and run performance tests
# Compare results across different states
```

### 4. Training and Demos

**Problem:** Need consistent demo environment.

```bash
# Create checkpoints for demo scenarios
# Replay to clean state after each demo
# Show time-travel capabilities
```

## Advanced Features

### Multiple Sessions

```bash
# Create session for feature A
SESSION_A=$(./examples/cli.py session-create "Feature A" | jq -r '.id')

# Create session for feature B
SESSION_B=$(./examples/cli.py session-create "Feature B" | jq -r '.id')

# Switch between sessions
./examples/cli.py session-switch $SESSION_A
./examples/cli.py session-switch $SESSION_B
```

### Backup and Restore

```bash
# Create backup
./postgres-test-replay -mode backup

# List available backups
ls -lh backups/

# Restore to replica (if needed)
./postgres-test-replay -mode restore \
  -backup backups/testdb_20240105_120000.sql \
  -target-db testdb
```

### Monitoring WAL Logs

```bash
# View raw WAL logs
tail -f waldata/wal_*.log | jq

# Count operations by type
grep -h "operation" waldata/wal_*.log | \
  jq -r '.operation' | \
  sort | uniq -c
```

## Troubleshooting

### WAL Listener Not Capturing Changes

**Check:**
1. Replication slot exists:
   ```sql
   SELECT * FROM pg_replication_slots;
   ```

2. Publication exists:
   ```sql
   SELECT * FROM pg_publication;
   ```

3. WAL level is logical:
   ```sql
   SHOW wal_level;
   ```

**Fix:**
```bash
# Restart listener
# It will try to create slot if missing
./postgres-test-replay -mode listener
```

### IPC Server Not Responding

**Check:**
```bash
# Verify server is running
curl http://localhost:8080/health

# Check logs
# Server prints startup message
```

### No WAL Entries Being Written

**Verify:**
1. Listener is running
2. Changes are being made to tables in publication
3. WAL directory has write permissions
4. Check listener logs for errors

### PostgreSQL Not Starting

**Check Docker:**
```bash
# View logs
make docker-logs

# Restart containers
make docker-down
make docker-up
```

## Tips and Best Practices

1. **Create checkpoints frequently** - Especially before risky operations

2. **Use descriptive names** - Makes navigation easier later

3. **One session per test scenario** - Keeps things organized

4. **Monitor disk space** - WAL logs can grow large

5. **Clean up old sessions** - Delete when no longer needed

6. **Backup before major changes** - Create full database backup

7. **Use the Python CLI** - Easier than raw curl commands

8. **Test replay in non-prod first** - Verify functionality works

## Configuration Tips

### Custom Ports

Edit `config.json`:
```json
{
  "primary_db": {
    "port": 5432
  },
  "replica_db": {
    "port": 5433
  }
}
```

### Custom Storage Paths

```json
{
  "storage": {
    "wal_log_path": "/data/waldata",
    "backup_path": "/data/backups",
    "session_path": "/data/sessions",
    "checkpoint_path": "/data/checkpoints"
  }
}
```

### Multiple Databases

Create separate sessions for each database:
```bash
./examples/cli.py session-create "DB1 Session" --database db1
./examples/cli.py session-create "DB2 Session" --database db2
```

## Performance Considerations

- WAL logs are append-only (fast writes)
- Reading all entries can be slow with large logs
- Consider rotating/archiving old logs
- Checkpoints help limit data processing
- Session replay may take time for many operations

## Security Notes

- Don't expose IPC server publicly
- Use firewall rules to restrict access
- Secure database credentials in config
- Don't commit config.json to git
- Regular backup of checkpoint/session data

## Next Steps

- Read [API.md](API.md) for complete API reference
- Read [CONTRIBUTING.md](CONTRIBUTING.md) to contribute
- Check GitHub issues for known problems
- Join discussions for questions

## Getting Help

- GitHub Issues: Report bugs or request features
- Documentation: Check README.md and API.md
- Examples: Review examples/ directory
- Tests: See test files for usage examples
