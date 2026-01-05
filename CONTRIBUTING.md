# Contributing to postgres-test-replay

Thank you for your interest in contributing to postgres-test-replay! This document provides guidelines and information for contributors.

## Getting Started

### Prerequisites

- Go 1.19 or higher
- Docker and Docker Compose
- PostgreSQL client tools (pg_dump, pg_restore)
- Git

### Setting Up Development Environment

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/postgres-test-replay.git
   cd postgres-test-replay
   ```

3. Install dependencies:
   ```bash
   make install
   ```

4. Build the project:
   ```bash
   make build
   ```

5. Start PostgreSQL:
   ```bash
   make docker-up
   ```

## Development Workflow

### Making Changes

1. Create a new branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes

3. Format your code:
   ```bash
   make fmt
   ```

4. Run linter:
   ```bash
   make lint
   ```

5. Build and test:
   ```bash
   make build
   make test
   ```

### Testing

Run tests with:
```bash
go test ./...
```

For verbose output:
```bash
go test -v ./...
```

### Code Style

- Follow standard Go conventions
- Use `gofmt` to format code
- Write clear, descriptive variable and function names
- Add comments for exported functions and types
- Keep functions focused and single-purpose

### Commit Messages

Write clear commit messages following this format:

```
Short summary (50 chars or less)

More detailed explanation if needed. Wrap at 72 characters.
Explain the problem this commit solves and why the change is needed.

- Bullet points are okay
- Use present tense ("Add feature" not "Added feature")
- Reference issues: Fixes #123
```

## Project Structure

```
.
├── cmd/                    # Application entry points
│   └── postgres-test-replay/
├── pkg/                    # Library packages
│   ├── backup/            # Backup/restore functionality
│   ├── checkpoint/        # Checkpoint management
│   ├── config/            # Configuration
│   ├── ipc/               # IPC server
│   ├── replication/       # WAL replication
│   ├── session/           # Session management
│   └── wal/               # WAL log handling
├── examples/              # Example scripts
├── docker-compose.yml     # Development environment
└── README.md
```

## Adding Features

### Adding a New Package

1. Create directory under `pkg/`
2. Add package documentation
3. Write tests
4. Update README if needed

### Extending the API

1. Add endpoint handler in `pkg/ipc/server.go`
2. Update API documentation in `API.md`
3. Add example usage
4. Test the endpoint

### Adding Configuration Options

1. Update `pkg/config/config.go`
2. Update `config.example.json`
3. Document in README

## Testing Guidelines

### Unit Tests

- Test files should be named `*_test.go`
- Use table-driven tests when appropriate
- Mock external dependencies
- Aim for good coverage of critical paths

Example:
```go
func TestBackupManager_CreateBackup(t *testing.T) {
    tests := []struct {
        name    string
        dbName  string
        wantErr bool
    }{
        {"valid backup", "testdb", false},
        {"empty name", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

### Integration Tests

- Use Docker for database setup
- Clean up resources after tests
- Test end-to-end workflows

## Documentation

### Code Documentation

- Document all exported functions, types, and constants
- Use godoc format
- Provide examples for complex functions

Example:
```go
// CreateBackup creates a backup of the specified database.
// It returns the path to the backup file or an error if the backup fails.
//
// Example:
//   backupFile, err := bm.CreateBackup(ctx, "mydb")
//   if err != nil {
//       log.Fatal(err)
//   }
func (bm *BackupManager) CreateBackup(ctx context.Context, dbName string) (string, error) {
    // implementation
}
```

### User Documentation

- Update README.md for user-facing changes
- Update API.md for API changes
- Add examples in the examples/ directory
- Include troubleshooting tips

## Pull Request Process

1. Update documentation
2. Add tests for new features
3. Ensure all tests pass
4. Update CHANGELOG if applicable
5. Create pull request with clear description
6. Address review comments
7. Wait for approval and merge

### Pull Request Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Manual testing completed

## Checklist
- [ ] Code follows project style
- [ ] Documentation updated
- [ ] Tests added/updated
- [ ] All tests pass
```

## Issue Reporting

### Bug Reports

Include:
- Clear, descriptive title
- Steps to reproduce
- Expected behavior
- Actual behavior
- Environment details (OS, Go version, etc.)
- Relevant logs or error messages

### Feature Requests

Include:
- Clear, descriptive title
- Problem the feature solves
- Proposed solution
- Alternative solutions considered
- Additional context

## Code Review

### As a Reviewer

- Be respectful and constructive
- Focus on the code, not the person
- Explain your reasoning
- Suggest alternatives
- Approve when ready

### As an Author

- Be open to feedback
- Ask questions if unclear
- Make requested changes promptly
- Thank reviewers for their time

## Release Process

1. Update version in relevant files
2. Update CHANGELOG
3. Create release branch
4. Test thoroughly
5. Create and tag release
6. Update documentation

## Getting Help

- Open an issue for bugs or features
- Check existing issues first
- Be patient and respectful
- Provide context and details

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

## Code of Conduct

### Our Standards

- Be respectful and inclusive
- Welcome diverse perspectives
- Focus on constructive feedback
- Support fellow contributors
- Maintain professionalism

### Enforcement

Violations may result in:
- Warning
- Temporary ban
- Permanent ban

Contact project maintainers to report issues.

## Recognition

Contributors will be recognized in:
- CONTRIBUTORS.md
- Release notes
- Project documentation

Thank you for contributing to postgres-test-replay!
