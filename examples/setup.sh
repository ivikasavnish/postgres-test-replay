#!/bin/bash
set -e

echo "=== PostgreSQL Test Replay Example ==="
echo

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}→ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

# Check if binary exists
if [ ! -f "./postgres-test-replay" ]; then
    print_info "Building application..."
    make build
    print_success "Build completed"
fi

# Create config if it doesn't exist
if [ ! -f "config.json" ]; then
    print_info "Creating configuration file..."
    cp config.example.json config.json
    print_success "Configuration created"
fi

# Start Docker Compose
print_info "Starting PostgreSQL containers..."
docker-compose up -d
sleep 5
print_success "PostgreSQL containers started"

# Wait for PostgreSQL to be ready
print_info "Waiting for PostgreSQL to be ready..."
for i in {1..30}; do
    if docker exec postgres-primary pg_isready -U postgres > /dev/null 2>&1; then
        print_success "PostgreSQL is ready"
        break
    fi
    if [ $i -eq 30 ]; then
        print_error "PostgreSQL failed to start"
        exit 1
    fi
    sleep 1
done

# Create some test data
print_info "Creating test data..."
docker exec postgres-primary psql -U postgres -d testdb -c "
INSERT INTO test_table (name, value) VALUES 
    ('test1', 100),
    ('test2', 200),
    ('test3', 300);
" > /dev/null 2>&1
print_success "Test data created"

# Create backup
print_info "Creating database backup..."
./postgres-test-replay -mode backup
print_success "Backup created"

# List backup files
print_info "Available backups:"
ls -lh backups/ 2>/dev/null || echo "  (none)"

echo
print_success "Setup completed successfully!"
echo
print_info "Next steps:"
echo "  1. Start the WAL listener: ./postgres-test-replay -mode listener"
echo "  2. Start the IPC server:   ./postgres-test-replay -mode ipc -addr :8080"
echo "  3. Make some database changes"
echo "  4. Use the API to create checkpoints and sessions"
echo
print_info "API Examples:"
echo "  # Create session"
echo '  curl -X POST http://localhost:8080/api/sessions -H "Content-Type: application/json" -d '"'"'{"name":"Test Session","description":"Example","database":"testdb"}'"'"
echo
echo "  # List sessions"
echo "  curl http://localhost:8080/api/sessions"
echo
echo "  # Health check"
echo "  curl http://localhost:8080/health"
echo
print_info "To stop everything:"
echo "  make docker-down"
