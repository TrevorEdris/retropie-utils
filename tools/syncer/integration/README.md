# Syncer Integration Tests

This directory contains comprehensive integration tests for the syncer tool using Docker (via testcontainers) and LocalStack to simulate AWS S3.

## Overview

The integration test suite covers:

1. **Sync Command Tests** (`sync_integration_test.go`)
   - Syncing ROMs, saves, and states individually
   - Syncing all file types together
   - Handling empty directories
   - Preserving directory structure in S3
   - Automatic bucket creation

2. **Config Command Tests** (`config_integration_test.go`)
   - Config validation with valid/invalid configurations
   - Username validation (length, characters, default username)
   - Config file creation and validation
   - Example config generation

3. **CLI Integration Tests** (`cli_integration_test.go`)
   - End-to-end testing of CLI commands
   - Testing sync command with valid/invalid configs
   - Testing config validation command
   - Testing init command

## Test Infrastructure

### Testcontainers

The tests use [testcontainers-go](https://github.com/testcontainers/testcontainers-go) to automatically manage LocalStack containers. Each test suite:
- Starts a LocalStack container before tests
- Configures AWS environment variables to point to LocalStack
- Creates necessary test data (ROMs, saves, states)
- Cleans up containers and test data after tests

### LocalStack

LocalStack emulates AWS S3 locally, allowing tests to run without requiring real AWS credentials or resources.

## Running Tests

### Using Make (Recommended)

```bash
# Run integration tests (uses testcontainers, no docker-compose needed)
make test-integration

# Alternative: Run with docker-compose
make test-integration-docker

# Start integration test environment manually
make integration-up

# Stop integration test environment
make integration-down
```

### Using Ginkgo Directly

```bash
# Run all integration tests
ginkgo -v ./tools/syncer/integration/...

# Run specific test file
ginkgo -v ./tools/syncer/integration/sync_integration_test.go
```

### Using Go Test

```bash
# Run all integration tests
go test -v ./tools/syncer/integration/...

# Run with coverage
go test -v -cover ./tools/syncer/integration/...
```

## Test Structure

```
integration/
├── README.md                      # This file
├── syncer_integration_suite_test.go  # Test suite entry point
├── helpers.go                     # Test helper functions
├── sync_integration_test.go       # Sync command tests
├── config_integration_test.go     # Config command tests
└── cli_integration_test.go        # CLI integration tests
```

## Helper Functions

The `helpers.go` file provides utilities for:
- Setting up and tearing down test environments
- Creating LocalStack containers
- Managing S3 buckets and objects
- Creating test files and directories
- Configuring AWS clients for LocalStack

## Requirements

- Docker (for testcontainers)
- Go 1.23+
- Ginkgo v2
- Gomega

## Notes

- Tests automatically clean up containers and temporary files
- Each test runs in isolation with its own LocalStack container
- Tests use temporary directories that are cleaned up after execution
- The test bucket name is `test-retropie-bucket` (configurable in helpers.go)

## Troubleshooting

### Docker Issues

If tests fail with Docker-related errors:
- Ensure Docker is running
- Check that you have permissions to run Docker
- Verify testcontainers can access Docker socket

### LocalStack Startup

If LocalStack fails to start:
- Check Docker logs: `docker ps -a`
- Increase timeout in `helpers.go` if needed
- Ensure port 4566 is not in use

### Test Failures

If tests fail:
- Check that all dependencies are installed: `go mod download`
- Verify LocalStack container is healthy
- Review test output for specific error messages

