# IWebhook Documentation

## Introduction

This webhook system is an application developed in Go that enables communication between internal services through events. The project implements an Event-Driven Architecture that allows registration, triggering, and processing of events between different services.

### Objectives

#### Implementation simplicity
- Webhooks basically work as HTTP requests between systems.
- No need to configure brokers, topics, or clusters like in Kafka.
- Ideal for quick and simple integrations between two or few systems.

#### Low operational cost
- No need for complex infrastructure.
- Avoids costs and effort with maintenance, scalability, and monitoring of a Kafka cluster.

#### Direct and instant communication
- The producer service immediately sends the HTTP request to the consumer.
- Ideal for events that need to be processed in real-time, with low latency.

#### Based on widely supported standards
- HTTP, REST, and JSON are known standards and widely supported by virtually all languages and frameworks.
- Avoids the need for specific Kafka client libraries.

#### Easy debugging and monitoring
- As it involves HTTP calls, it's easy to capture logs, use tools like Postman, cURL, or proxies (e.g., ngrok) for testing.
- Doesn't require specialized inspection tools like Kafka needs (e.g., Kafka Tool, KSQL, etc.).

#### Decoupling via endpoints
- Each system only needs to expose or consume an HTTP endpoint, without worrying about queues, partitions, or consumer groups.

#### Less dependency on technology stack
- Webhooks can be implemented even in legacy or monolithic systems that don't have the capacity or need for Kafka adoption.

---

## System Architecture

### Architecture Overview

```mermaid
graph TB
    %% Serviços
    Services[Serviços]
    
    %% Sistema Webhook Central
    subgraph "Sistema Webhook"
        API[API REST]
        Worker[Worker]
        Database[(Banco de Dados)]
        Queue[(Fila)]
    end
    
    %% Consumidores
    Consumers[Serviços Consumidores]
    
    %% Fluxos
    Services -->|Criar Eventos<br/>Registrar Gatilhos<br/>Publicar Eventos| API
    API --> Database
    API --> Queue
    Queue --> Worker
    Worker --> Database
    Worker -->|Webhooks HTTP| Consumers
    
    %% Estilos
    classDef serviceStyle fill:#e3f2fd,stroke:#1976d2,stroke-width:2px
    classDef systemStyle fill:#f3e5f5,stroke:#7b1fa2,stroke-width:2px
    classDef storageStyle fill:#e8f5e8,stroke:#388e3c,stroke-width:2px
    
    class Services,Consumers serviceStyle
    class API,Worker systemStyle
    class Database,Queue storageStyle
```

---

## Installation and Usage

### 🐳 Using Docker Hub

The Docker image is available on Docker Hub for easy use:

**Docker Image**: `isaacdsc/gqueue:latest`

**Docker Hub Link**: [https://hub.docker.com/repository/docker/isaacdsc/gqueue/general](https://hub.docker.com/repository/docker/isaacdsc/gqueue/general)

#### Quick Usage with Docker

```bash
# Pull the image from Docker Hub
docker pull isaacdsc/gqueue:latest

# Run with dependencies using docker-compose
docker compose up
```

#### Configuration with Docker Compose

You can use the `compose.yaml` file included in the project that is already configured with the Docker Hub image and all dependencies (Redis and MongoDB):

```yaml
services:
  server:
    image: isaacdsc/gqueue:latest  # Using the Docker Hub image
    ports:
      - 8080:8080
    depends_on:
      - redis
      - mongodb
    environment:
      - DB_CONNECTION_STRING=mongodb://root:example@mongodb:27017/
      - CACHE_ADDR=redis:6379
      # ... other environment variables
```

#### Main Environment Variables

- `DB_CONNECTION_STRING`: MongoDB connection string
- `CACHE_ADDR`: Redis address
- `WQ_CONCURRENCY`: Number of simultaneous workers
- `WQ_QUEUES`: Queue configuration (JSON)

For more Docker details, see: [README.Docker.md](README.Docker.md)

<!-- **Trigger Types:**

----

## Security Recommendations

### ⚠️ Private Network Communication

**IMPORTANT**: To ensure secure communication between services, it is **highly recommended** that:

- **All services using the webhook system are executed in a private/internal network**
- **Webhook endpoints should NOT be exposed publicly on the internet**
- **Use virtual private networks (VPN) or internal networks (such as Docker networks, internal Kubernetes clusters)**

### Recommended Practices

1. **Network Isolation**: Configure firewalls and security groups to allow communication only between authorized services
2. **Authentication**: Implement authentication mechanisms between services (tokens, certificates, etc.)
3. **Encryption**: Use HTTPS/TLS for communication between services, even on internal networks
4. **Monitoring**: Implement audit logs to track all webhook communications
5. **Rate Limiting**: Configure rate limits to prevent endpoint abuse

---

