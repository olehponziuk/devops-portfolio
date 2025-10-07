# Git Hooks Demo

This repository demonstrates Git hooks for automated linting, testing, and backup.

## Git Hooks

### Pre-commit Hook
- **Purpose**: Runs linting before each commit
- **Commands**: `go fmt ./...` and `go vet ./...`
- **Behavior**: Blocks commit if linting fails

### Pre-push Hook  
- **Purpose**: Runs tests before each push
- **Commands**: `go test ./...`
- **Behavior**: Blocks push if tests fail

### Post-commit Hook
- **Purpose**: Creates automatic backups after each commit
- **Commands**: Creates timestamped Git bundle in `../backup/`
- **Behavior**: Runs automatically after successful commits

## Setup

To install the Git hooks on your machine:

```bash
./setup-hooks.sh
```

This will copy the hooks from the `hooks/` directory to `.git/hooks/` and make them executable.

## 📁 Files

- `hooks/` - Contains the Git hook scripts
- `setup-hooks.sh` - Installation script
- `main.go` - Sample Go application
- `main_test.go` - Sample Go tests
