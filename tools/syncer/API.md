# Syncer API

The Syncer API provides HTTP endpoints to trigger and monitor sync operations.

## Endpoints

### `GET /health`
Health check endpoint.

**Response:**
```json
{
  "status": "healthy"
}
```

### `POST /sync`
Triggers a sync operation. The operation runs asynchronously.

**Response (202 Accepted):**
```json
{
  "success": true,
  "message": "Sync operation started",
  "start_time": "2024-01-01T12:00:00Z"
}
```

**Response (409 Conflict):**
```json
{
  "success": false,
  "message": "Sync operation already in progress"
}
```

### `GET /status`
Returns the current status of the syncer.

**Response:**
```json
{
  "is_running": false,
  "last_sync_time": "2024-01-01T12:00:00Z",
  "last_error": "error message if any"
}
```

## Usage Examples

### Trigger a sync operation
```bash
curl -X POST http://localhost:8000/sync
```

### Check sync status
```bash
curl http://localhost:8000/status
```

### Health check
```bash
curl http://localhost:8000/health
```

## Running the API Server

### Command Line
```bash
syncer api --port 8000 --config /path/to/config.yaml
```

### Docker Compose
The API server runs automatically when the syncer container starts. It listens on port 8000.

## Notes

- Sync operations run asynchronously - the API returns immediately after starting the operation
- Only one sync operation can run at a time
- Use the `/status` endpoint to check if a sync is in progress or view the last sync result
- All endpoints return JSON responses

