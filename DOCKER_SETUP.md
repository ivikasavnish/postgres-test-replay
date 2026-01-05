# Docker Compose Automation Setup

This document describes how to use the automated Docker Compose setup with backup management and port reservation.

## Features

1. **Automated Docker Compose Management**: No need to manually run `docker-compose up`. The setup script handles it.
2. **Timestamped Backups**: Automatically creates timestamped backups of `docker-compose.yml` before modifications.
3. **Backup Rotation**: Keeps only the 3 most recent backups to save disk space.
4. **Port Reservation**: Uses a reserved pool of ports (58180-58190) to avoid conflicts with common services.
5. **Dynamic Port Discovery**: Automatically finds available ports within the reserved pool.
6. **Automatic .env Generation**: Creates/updates the `.env` file with discovered DSNs and ports.
7. **UI DSN Display**: The web UI dynamically displays the current database connection strings.

## Reserved Port Pool

The system reserves ports **58180-58190** for use:
- Ports 58180-58187: Available for PostgreSQL primary and replica instances
- Ports 58188-58190: Available for the UI server

This range avoids common service ports (e.g., 8080, 3000, 5432) to prevent conflicts.

## Usage

### Step 1: Run the Setup Script

```bash
# Build the setup tool
go build -o pgparsers ./cmd/pgparsers

# Run the setup
./pgparsers
```

This will:
1. Backup the existing `docker-compose.yml` (if any)
2. Read the current docker-compose configuration
3. Discover available ports from the reserved pool
4. Update the docker-compose.yml with new ports
5. Generate/update the `.env` file
6. Stop any existing Docker Compose services
7. Start the new Docker Compose services

### Step 2: Run the Application

After the setup completes, it will display instructions like:

```
=== Setup Complete ===
Primary DSN: postgres://postgres:postgres@localhost:58184/voter-outreach-new?sslmode=disable
Replica DSN: postgres://postgres:postgres@localhost:58185/voter-outreach-new?sslmode=disable
Server will run on port: 58186

You can now run the application with:
  ./postgres-test-replay -mode ipc -addr :58186
```

Follow the instructions to start the IPC server:

```bash
# Build the main application
go build -o postgres-test-replay ./cmd/postgres-test-replay

# Run in IPC mode with the discovered port
./postgres-test-replay -mode ipc -addr :58186
```

### Step 3: Access the UI

Open your browser and navigate to:
```
http://localhost:58186
```

The UI will automatically display the current DSN configuration.

## Backup Management

Backups are stored in the `./backups` directory with timestamps:
```
backups/
  docker-compose_20240105_143022.yml
  docker-compose_20240105_150145.yml
  docker-compose_20240105_153317.yml
```

Only the 3 most recent backups are kept. Older backups are automatically deleted.

## Configuration Files

### .env File
Generated automatically with the following structure:
```env
# Database DSN Configuration
INPUT_DSN=postgres://postgres:postgres@localhost:58184/voter-outreach-new?sslmode=disable
OUTPUT_DSN=postgres://postgres:postgres@localhost:58185/voter-outreach-new?sslmode=disable

# Server Configuration
SERVER_PORT=58186
SERVER_UI_PATH=./ui

# Storage Configuration
WAL_LOG_PATH=./waldata
BACKUP_PATH=./backups
SESSION_PATH=./sessions
CHECKPOINT_PATH=./checkpoints

# Replication Configuration
REPLICATION_SLOT=test_slot
PUBLICATION_NAME=test_publication
```

### config.yaml File
Also generated automatically:
```yaml
primary_dsn: postgres://postgres:postgres@localhost:58184/voter-outreach-new?sslmode=disable
replica_dsn: postgres://postgres:postgres@localhost:58185/voter-outreach-new?sslmode=disable
```

## Docker Compose Integration

The setup script automatically:
- Stops existing services: `docker-compose down -v`
- Starts new services: `docker-compose up -d`
- Waits for services to be ready

## Troubleshooting

### Port Already in Use
If all ports in the reserved pool are in use, you'll see an error:
```
panic: no available ports in pool 58180-58190
```

Solution: Stop other services using those ports or increase the port pool range in `cmd/pgparsers/main.go`.

### Docker Compose Not Found
If docker-compose is not installed:
```
Failed to start Docker Compose services: docker-compose up failed: exec: "docker-compose": executable file not found
```

Solution: Install Docker and Docker Compose.

### Services Not Starting
Check Docker logs:
```bash
docker-compose logs
```

## Manual Override

If you need to manually manage Docker Compose:

```bash
# Stop services
docker-compose down -v

# Start services
docker-compose up -d

# View logs
docker-compose logs -f
```

## Testing

Run the test suite to verify functionality:

```bash
cd cmd/pgparsers
go test -v
```

Tests cover:
- Backup file creation and rotation
- Port discovery from reserved pool
- .env file generation
