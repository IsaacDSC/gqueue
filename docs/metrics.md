# gqueue Metrics and Usage Manual

This document describes the metrics exposed by gqueue via OpenTelemetry (OTel) in Prometheus format, and provides a **usage manual** with commands to run services, example consumers, and load simulations.

## Overview

- **Format:** Prometheus exposition format.
- **Endpoint:** Each HTTP service exposes `GET /metrics` on the same port as the API.
- **Configuration:** Metrics can be disabled with `METRICS_ENABLED=false` (default: `true`).
- **Stack:** Prometheus scrapes the endpoints; Grafana uses the Prometheus datasource for visualization (Docker Compose profile `observability`).

## Metric endpoints by service

| Service   | Port | Metrics URL                 |
|----------|------|-----------------------------|
| Backoffice | 8081 | http://localhost:8081/metrics |
| PubSub API | 8082 | http://localhost:8082/metrics |
| Task API   | 8083 | http://localhost:8083/metrics |

The `/metrics` and `/health` routes do **not** generate HTTP metrics (they are excluded by the middleware to avoid noise).

---

## Usage manual

### Start the stack with observability

Start all services (Backoffice, PubSub, Task, PostgreSQL, Redis, etc.) plus Prometheus and Grafana:

```sh
docker compose -f deployment/app-pgsql/docker-compose.yaml --profile observability up -d --build
```

- **Grafana:** http://localhost:3000 (Prometheus datasource is provisioned).
- **Prometheus:** scrapes Backoffice (8081), PubSub (8082), and Task (8083) `/metrics`.

### Start the example consumer

Run the example consumer (e.g. to process PubSub or Task events) in the foreground:

```sh
docker compose -f deployment/app-pgsql/docker-compose.yaml --profile example up
```

### Generate load (simulate producers)

From the **project root**, run the simulation scripts to send traffic to the APIs.

**PubSub producers (POST to PubSub API):**

```sh
example/simulation/multiples_producer_pubsub.sh
```

**Task producers (POST to Task API):**

```sh
example/simulation/multiples_producer_task.sh
```

Ensure the stack is up (`observability` or at least the API services) before running these scripts.

---

## Metrics reference by category

### Backoffice metrics

| Metric / concept           | Description |
|----------------------------|-------------|
| **CPU**                    | Process CPU usage (e.g. from Go runtime / Prometheus `process_*` or host metrics). |
| **Memory**                 | Process memory usage (RSS, heap, etc.). |
| **Number of goroutines**   | Active goroutines (e.g. `go_goroutines` if exposed). |
| **Number of threads**      | OS threads used by the process. |
| **GC activity**            | Garbage collection activity (allocations, GC pauses). |
| **Total connections (Pg)** | Total number of PostgreSQL connections used by the Backoffice. |
| **Total GB on disk**       | Disk usage (e.g. PostgreSQL data directory or host disk). |
| **Cache DB ratio**         | Database cache hit ratio (e.g. PostgreSQL `cache hit ratio`). |
| **Disk DB reading**        | Disk read I/O for the database. |

HTTP metrics for the Backoffice (request count, duration histogram) are exposed via the metrics middleware; see [HTTP metrics](#http-metrics-middleware) below.

---

### PubSub metrics

| Metric / concept            | Description |
|-----------------------------|-------------|
| **P95 publisher msg**       | 95th percentile latency for publishing a message (publisher path). |
| **RPM publisher msg**       | Requests per minute for the publisher (messages published per minute). |
| **RPM consumer msg**        | Messages consumed per minute (consumer throughput). |
| **RPM client HTTP sent msg**| Rate of messages sent via HTTP client (publish API calls per minute). |
| **Lag consumer msg**        | Time between message publish and consumer processing (consumer lag). |
| **Mem store sync activity** | Mem store sync operations (e.g. flushes to persistent storage). |
| **Mem store time execution (ms)** | Mem store operation duration in milliseconds. |
| **CPU**                     | Process CPU usage. |
| **Memory**                  | Process memory usage. |
| **Number of goroutines**    | Active goroutines. |
| **Number of threads**       | OS threads. |
| **GC activity**             | Garbage collection activity. |

Consumer lag is exposed as `pubsub_consumer_lag_seconds` (histogram); see [PubSub consumer lag](#pubsub_consumer_lag_seconds) for PromQL examples.

---

### Task metrics

| Metric / concept            | Description |
|-----------------------------|-------------|
| **Total processing task**  | Total tasks currently being processed (or total processed over time). |
| **Total success consumer task** | Total tasks consumed successfully. |
| **Total failure consumer task** | Total tasks that failed consumption (e.g. retries or dead-letter). |
| **P95 publisher msg**      | 95th percentile latency for task publish (API). |
| **RPM task publisher**     | Task publish requests per minute. |
| **RPM task consumer**      | Tasks consumed per minute. |
| **RPM client HTTP sent msg**| Task API publish calls per minute. |
| **CPU**                    | Process CPU usage. |
| **Memory**                 | Process memory usage. |
| **Number of goroutines**    | Active goroutines. |
| **Number of threads**       | OS threads. |
| **GC activity**             | Garbage collection activity. |

Task queue gauges (`task_queue_pending`, `task_queue_active`, etc.) and consumer counters are described below.

---

## HTTP metrics (middleware)

Recorded automatically by the metrics middleware on all services (Backoffice, PubSub, Task). They follow Prometheus and OTel HTTP semantic conventions.

### `http_server_requests_total`

- **Type:** counter (`Int64Counter`).
- **Description:** Total HTTP requests received.
- **Labels:** `http.method`, `http.route`, `http.status_code`, `service.name` (`backoffice`, `pubsub`, `task`).

### `http_server_request_duration_seconds`

- **Type:** histogram (`Float64Histogram`).
- **Description:** Request duration in seconds.
- **Labels:** same as above.

In Prometheus, the histogram is exposed as:

- `http_server_request_duration_seconds_bucket`
- `http_server_request_duration_seconds_sum`
- `http_server_request_duration_seconds_count`

**Example (p95 by route and service):**

```promql
histogram_quantile(
  0.95,
  sum(rate(http_server_request_duration_seconds_bucket[5m])) by (le, http_route, service_name)
)
```

---

## Domain metrics

### Task: publisher and queue

- **`task_publisher_requests_total`** — Counter of requests to the task publish endpoint (Task API, `POST /api/v1/task`).

Queue metrics (Asynq / Task) represent real backlog in Redis; labels include `queue` and `service.name` = `task`:

| Metric               | Type  | Description |
|----------------------|-------|-------------|
| `task_queue_pending` | gauge | Tasks waiting to be processed. |
| `task_queue_active`  | gauge | Tasks currently being processed. |
| `task_queue_scheduled` | gauge | Tasks scheduled for the future. |
| `task_queue_retry`   | gauge | Tasks waiting for retry. |
| `task_queue_archived`| gauge | Tasks archived (e.g. dead-letter). |
| `task_queue_size`    | gauge | Total tasks in the queue. |
| `task_queue_latency_seconds` | gauge | Queue latency in seconds (age of oldest pending task). |

**Task consumer totals (dashboard queries):**

Success count by event name:

```promql
sum by (task_event_name) (task_consumer_success_total)
```

Failure count by event name:

```promql
sum by (task_event_name) (task_consumer_failure_total)
```

Processing (e.g. in-flight or total processed) by event name:

```promql
sum by (task_event_name) (task_consumer_total_processing)
```

---

### `pubsub_consumer_lag_seconds`

- **Type:** histogram (`Float64Histogram`).
- **Description:** Time in seconds between message publish and the start of processing by the consumer (message age at consume time).
- **Service:** PubSub API (consumer).
- **Labels:** `topic`, `consumer.service_name` (or `consumer_service_name` depending on exporter).

**Example (p99 lag by topic):**

```promql
histogram_quantile(
  0.99,
  sum(rate(pubsub_consumer_lag_seconds_bucket[5m])) by (le, topic, consumer_service_name)
)
```

**Example (p95 by topic):**

```promql
histogram_quantile(0.95, sum(rate(pubsub_consumer_lag_seconds_bucket[5m])) by (le, topic))
```

**Example (throughput — messages/second with lag):**

```promql
sum(rate(pubsub_consumer_lag_seconds_count[5m])) by (topic, consumer_service_name)
```

---

## Prometheus scrape configuration

In the Docker Compose deployment (profile `observability`), Prometheus is configured to scrape:

- **Backoffice:** `backoffice:8081/metrics`
- **PubSub:** `pubsub:8082/metrics`
- **Task:** `task:8083/metrics`

Configuration file: `deployment/app-pgsql/prometheus/prometheus.yml`.

---

## References

- Implementation spec: [docs/specs/opentelemetry-metrics.md](specs/opentelemetry-metrics.md).
- Metrics middleware: `cmd/setup/middleware/middleware.go` (MetricsMiddleware).
- Telemetry package: `pkg/telemetry`.
- [OpenTelemetry semantic conventions — HTTP](https://opentelemetry.io/docs/specs/semconv/http/).
