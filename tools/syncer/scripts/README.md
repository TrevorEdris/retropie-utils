# API Scripts

Shell scripts for interacting with the Syncer API.

## Available Scripts

- `health.sh` - Check API health status
- `sync.sh` - Trigger a sync operation
- `status.sh` - Get current sync status

## Usage

### Basic Usage

```bash
# Health check
./scripts/health.sh

# Trigger sync
./scripts/sync.sh

# Check status
./scripts/status.sh
```

### Custom API URL

Set the `API_URL` environment variable to use a different API endpoint:

```bash
export API_URL=http://localhost:8000
./scripts/sync.sh
```

Or inline:

```bash
API_URL=http://localhost:8000 ./scripts/sync.sh
```

## Examples

### Check if API is healthy
```bash
./scripts/health.sh
```

### Trigger a sync and check status
```bash
./scripts/sync.sh
sleep 2
./scripts/status.sh
```

### Monitor sync status
```bash
./scripts/sync.sh
while true; do
  ./scripts/status.sh
  sleep 1
done
```

## Exit Codes

- `0` - Success
- `1` - Error or failure

## Dependencies

- `curl` - HTTP client
- `jq` - JSON processor (optional, for pretty output)

