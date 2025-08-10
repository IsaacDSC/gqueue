# Docker Documentation

## 🐳 Using Pre-built Image from Docker Hub

### Available Image

**Docker Image**: `isaacdsc/gqueue:latest`

**Docker Hub**: [https://hub.docker.com/repository/docker/isaacdsc/gqueue/general](https://hub.docker.com/repository/docker/isaacdsc/gqueue/general)

### Quick Usage

```bash
# Pull the image from Docker Hub
docker pull isaacdsc/gqueue:latest

# Run with docker-compose (recommended)
docker compose up

# Or run only the application (requires external Redis and MongoDB)
docker run -p 8080:8080 \
  -e DB_CONNECTION_STRING="mongodb://root:example@localhost:27017/" \
  -e CACHE_ADDR="localhost:6379" \
  isaacdsc/gqueue:latest
```

### Configuration with Docker Compose

The `compose.yaml` file is already configured to use the Docker Hub image:

```yaml
services:
  server:
    image: isaacdsc/gqueue:latest  # Pre-built image
    ports:
      - 8080:8080
    depends_on:
      - redis
      - mongodb
    environment:
      - DB_CONNECTION_STRING=mongodb://root:example@mongodb:27017/
      - CACHE_ADDR=redis:6379
      - WQ_CONCURRENCY=32
      - WQ_QUEUES='{"internal.critical": 7, "internal.high": 5, "internal.medium": 3, "internal.low": 1, "external.critical": 7, "external.high": 5, "external.medium": 3, "external.low": 1}'
```

---

## Building Your Own Image (Development)

### Building and running your application

When you're ready, start your application by running:
`docker compose up --build`.

Your application will be available at http://localhost:8080.

### Deploying your application to the cloud

First, build your image, e.g.: `docker build -t myapp .`.
If your cloud uses a different CPU architecture than your development
machine (e.g., you are on a Mac M1 and your cloud provider is amd64),
you'll want to build the image for that platform, e.g.:
`docker build --platform=linux/amd64 -t myapp .`.

Then, push it to your registry, e.g. `docker push myregistry.com/myapp`.

Consult Docker's [getting started](https://docs.docker.com/go/get-started-sharing/)
docs for more detail on building and pushing.

### References
* [Docker's Go guide](https://docs.docker.com/language/golang/)