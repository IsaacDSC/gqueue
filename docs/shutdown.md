# Graceful Shutdown in gqueue

## Overview
This document describes the centralized graceful shutdown in gqueue: how it is triggered, how HTTP servers and workers are stopped, and how to test it.

## How Shutdown Works

1. **Signal handling**  
   In `cmd/api/main.go`, `waitForShutdown` listens for `SIGINT` and `SIGTERM` (and for `ctx.Done()` if the context is cancelled). The main process does not exit until shutdown has finished.

2. **HTTP servers**  
   When a signal is received, the code logs `"Shutting down servers..."` and then calls `server.Shutdown(shutdownCtx)` for each HTTP server (Backoffice, PubSub API, Task API). Shutdown uses a **15 second** timeout so in-flight requests can finish. Servers are shut down in sequence.

3. **Closers**  
   After all servers have shut down, `main` runs the registered closers (e.g. `pubsub.Close`, `task.Close`). Those closers stop workers and release resources. Only then does the process exit.

4. **Context**  
   The global `context.Context` created in `main` is passed into PubSub and Task when they start. Workers and long-running logic should respect this context so they can stop when the process is shutting down (typically as a result of closers running).

## Configuration

- **Shutdown timeout**  
  The timeout used in `waitForShutdown` for `server.Shutdown` is currently **15 seconds** (hardcoded in `cmd/api/main.go`). The config field `ShutdownTimeout` (`env:"SHUTDOWN_TIMEOUT"`, default `30s`) exists in `internal/cfg` but is not yet used there; it can be wired in later if desired.

- **Concurrency**  
  Worker concurrency for PubSub and Task is controlled by `AsynqConfig.Concurrency` / `WQ_CONCURRENCY` (see `internal/cfg`).

## Services and Workers

- **Backoffice HTTP server**  
  Started via `backoffice.Start(...)`. Shut down by the central handler with the same 15s timeout.

- **PubSub**  
  Started via `pubsub.New(...).Start(ctx, conf)`. The PubSub HTTP server is in the `servers` list and is shut down by `waitForShutdown`. Subscribers and other resources are stopped when the PubSub closer runs.

- **Task (Asynq)**  
  Started via `task.New(...).Start(ctx, conf)`. The Task API server is shut down by `waitForShutdown`; the Asynq worker and related resources are stopped when the Task closer runs.

## Testing

An integration test checks that shutdown completes correctly when the process receives SIGINT:

- **Test:** `cmd/api/shutdown_test.go` — `TestShutdownGraceful`
- **What it does:** Builds the API binary, starts it, holds an in-flight HTTP connection to the backoffice server, sends SIGINT, then checks that the process exits with code 0, that the logs show shutdown started and completed, and that shutdown took at least 2 seconds (so the server actually waited for the in-flight connection).
- **Env:** The test skips if `DB_CONNECTION_STRING` or `CACHE_ADDR` are not set. It loads `.env` from the module root (stdlib only, no extra deps) so you can run `go test` with env from a local `.env`.
- **Run:**
  - From repo root with env already set (e.g. `.env` or direnv):  
    `go test -v -run TestShutdownGraceful ./cmd/api/`
  - Or use the script that sources `.env-example` and `.env` then runs the test:  
    `./scripts/run_shutdown_test.sh`

Requires Redis and Postgres (and other env) to be up so the binary can start.

## Best Practices

- Pass the context from `main` into any new goroutine or handler so they can stop when the process is shutting down.
- Rely on the central shutdown in `main`; avoid extra signal handling or ad-hoc shutdown logic in services.
- Prefer operations that can be interrupted via context (e.g. context-aware HTTP clients, `ctx.Done()` in loops).

## References

- `cmd/api/main.go` — `waitForShutdown`, `main` (servers, closers, and scope).
- `cmd/setup/pubsub` and `cmd/setup/task` — how services register servers and closers and use the context.
- `cmd/api/shutdown_test.go` — integration test for graceful shutdown.
