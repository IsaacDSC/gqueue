# Docker Documentation

## Using Pre-built Image from Docker Hub

### Available Image

**Docker Image**: `isaacdsc/gqueue:latest`

**Docker Hub**: [https://hub.docker.com/repository/docker/isaacdsc/gqueue/general](https://hub.docker.com/repository/docker/isaacdsc/gqueue/general)

### Quick Usage

```bash
# Pull the image from Docker Hub
docker pull isaacdsc/gqueue:latest

# Run with docker-compose (recommended) — see "Configuration with Docker Compose" below
docker compose -f deployment/app-pgsql/docker-compose.yaml --profile complete up -d

# Or run only the application (requires external Postgres and Redis)
# By default all scopes run in one container. Use --scope to run a single scope:
docker run -p 8081:8081 \
  -e DB_DRIVER=pg \
  -e DB_CONNECTION_STRING="postgres://user:pass@host:5432/gqueue?sslmode=disable" \
  -e CACHE_ADDR="host:6379" \
  isaacdsc/gqueue:latest --scope=backoffice

# Other scopes: --scope=pubsub (port 8082), --scope=task (port 8083), or omit for --scope=all (ports 8081, 8082, 8083)
```

### Configuration with Docker Compose

The application runs as three separate services (scopes): **backoffice**, **pubsub**, and **task**. The compose file is at `deployment/app-pgsql/docker-compose.yaml`.

- **backoffice** — port 8081, profile `complete`
- **pubsub** — port 8082, profile `complete` (depends on pubsub-emulator)
- **task** — port 8083, profile `complete` (depends on pubsub-emulator)

Profiles:

- **infra**: postgres, redis, pubsub-emulator
- **complete**: backoffice, pubsub, task (and infra when brought up together)
- **debug**: adds PgAdmin at http://localhost:5050
- **example**: example consumer (port 3333) for receiving webhooks from pubsub/task

**Run all three scopes plus the example consumer** (so pubsub/task can notify the consumer):

```bash
docker compose -f deployment/app-pgsql/docker-compose.yaml --profile complete --profile example up -d
```

From inside the **pubsub** and **task** containers, the consumer is reachable at **`http://consumer:3333`** (Docker service name). When registering a consumer in the backoffice (event consumer URL), use that base URL, e.g. `http://consumer:3333` and path `/` (the example consumer accepts `POST /`). No host mount is needed—containers share the same network.

**Run only infrastructure:**

```bash
docker compose -f deployment/app-pgsql/docker-compose.yaml --profile infra up -d
```

**Run a single scope** (e.g. backoffice only):

```bash
docker compose -f deployment/app-pgsql/docker-compose.yaml --profile infra --profile complete up -d backoffice
```

**Run all three scopes:**

```bash
docker compose -f deployment/app-pgsql/docker-compose.yaml --profile complete up -d
```

**Endpoints:**

- Backoffice: http://localhost:8081
- Pubsub: http://localhost:8082
- Task: http://localhost:8083
- PgAdmin (with `--profile debug`): http://localhost:5050

---

## Building Your Own Image (Development)

### Building and running your application

From the project root, build and run using the app-pgsql compose file:

```bash
docker compose -f deployment/app-pgsql/docker-compose.yaml --profile complete up --build
```

Your application is exposed as three services:

- **Backoffice**: http://localhost:8081
- **Pubsub**: http://localhost:8082
- **Task**: http://localhost:8083

To run only one scope, start infra first then the desired service, e.g.:

```bash
docker compose -f deployment/app-pgsql/docker-compose.yaml --profile infra --profile complete up -d --build backoffice
```

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
