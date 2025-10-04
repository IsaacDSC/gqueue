# Consumer Webhook Example

This is a simple webhook consumer application that demonstrates how to receive and process webhook events from the gqueue system.

## Overview

The consumer is a lightweight HTTP server written in Go that:

- Listens on port 3333
- Accepts POST requests to any endpoint
- Logs all incoming request headers and body content
- Returns a simple "Hello, Webhook!" response

## Features

- **Simple HTTP Server**: Built using Go's standard `net/http` package
- **Request Logging**: Logs headers, method, path, and request body
- **Docker Support**: Includes Dockerfile for containerized deployment
- **Docker Compose Integration**: Part of the infrastructure profile

## Running the Consumer

### Using Docker Compose (Recommended)

The consumer is included in the docker-compose setup with the `infra` profile:

```bash
# Navigate to the deployment directory
cd deployment/app-pgsql

# Start infrastructure services including the consumer
docker-compose --profile test up -d

# Check service status
docker-compose --profile test ps

# View consumer logs
docker-compose logs consumer

# Stop services
docker-compose --profile test down
```

### Using Docker

```bash
# Build the consumer image
docker build -t consumer example/consumer/

# Run the container
docker run -p 3333:3333 consumer
```

### Running Locally

```bash
# Navigate to consumer directory
cd example/consumer

# Run the application
go run main.go
```

## Testing the Consumer

Once the consumer is running, you can test it by sending HTTP POST requests:

```bash
# Simple test
curl -X POST http://localhost:3333 \
  -d '{"message": "Hello, Webhook!"}' \
  -H "Content-Type: application/json"

# Test with more complex data
curl -X POST http://localhost:3333 \
  -d '{"user": "john", "action": "login", "timestamp": "2024-01-01T10:00:00Z"}' \
  -H "Content-Type: application/json"

# Test with different endpoints
curl -X POST http://localhost:3333/notifications/new-user \
  -d '{"user_id": 123, "email": "user@example.com"}' \
  -H "Content-Type: application/json"
```

## Expected Output

When you send a request to the consumer, you'll see:

**Response:**

```
Hello, Webhook!
```

**Logs:**

```
[*] Received Headers: map[Accept:[*/*] Content-Length:[30] Content-Type:[application/json] User-Agent:[curl/8.7.1]]
[*] Received request: POST /
{"message": "Hello, Webhook!"}
```

## Configuration

The consumer currently has minimal configuration:

- **Port**: 3333 (hardcoded)
- **Endpoints**: Accepts POST requests to any path
- **Response**: Always returns "Hello, Webhook!"

## Docker Configuration

### Dockerfile Features

- **Multi-stage build**: Separates build and runtime environments
- **Minimal base image**: Uses Alpine Linux for small image size
- **Non-root user**: Runs with limited privileges for security
- **Health check ready**: Exposes port 3333 for health monitoring

### Docker Compose Integration

The consumer service includes:

- **Profile**: `infra` - starts with infrastructure services
- **Network**: Connected to `app-network` for service communication
- **Port mapping**: `3333:3333` for external access
- **Resource limits**: CPU and memory constraints for stable operation

## Use Cases

This consumer example can be used to:

1. **Test webhook delivery**: Verify that gqueue can successfully deliver webhooks
2. **Debug webhook payloads**: Inspect the structure and content of webhook events
3. **Prototype webhook handlers**: Use as a starting point for custom webhook processing
4. **Load testing**: Test webhook delivery performance under load

## Extending the Consumer

To customize the consumer for your needs, you can modify `main.go` to:

- Add authentication/authorization
- Route different webhook types to different handlers
- Store webhook data in a database
- Forward webhooks to other services
- Implement custom response logic
- Add metrics and monitoring

## Troubleshooting

### Container won't start

- Check if port 3333 is already in use: `lsof -i :3333`
- Verify Docker is running: `docker ps`
- Check logs: `docker-compose logs consumer`

### Can't reach the endpoint

- Verify the service is running: `docker-compose ps`
- Check port mapping: `localhost:3333` should be accessible
- Test with curl or a similar tool

### No logs appearing

- Ensure you're using POST requests (GET requests will return 405)
- Check that the request body is not empty
- Verify the container is receiving traffic: `docker-compose logs consumer`
