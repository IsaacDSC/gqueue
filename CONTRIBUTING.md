# Contributing to IWebhook

We love your input! We want to make contributing to this webhook system as easy and transparent as possible, whether it's:

- Reporting a bug
- Discussing the current state of the code
- Submitting a fix
- Proposing new features
- Becoming a maintainer

## 🚀 Development Setup

### Prerequisites

- **Go**: Version 1.23.7 or higher
- **Docker & Docker Compose**: For running dependencies (Redis, MongoDB)
- **Make**: For using the Makefile commands
- **Git**: For version control

### Getting Started

1. **Fork and Clone the Repository**
   ```bash
   git clone https://github.com/your-username/gqueue.git
   cd gqueue
   ```

2. **Install Dependencies**
   ```bash
   go mod download
   ```

3. **Start Dependencies with Docker**
   ```bash
   docker compose up -d redis mongodb
   ```

4. **Run the Application**
   ```bash
   # Start all services (webhook API + worker)
   make run-all
   
   # Or start services separately
   make run-webhook  # API only
   make run-worker   # Worker only
   ```

### Available Make Commands

| Command | Description |
|---------|-------------|
| `make build` | Build the application binary |
| `make run-all` | Start both webhook API and worker services |
| `make run-webhook` | Start only the webhook API service |
| `make run-worker` | Start only the worker service |
| `make test` | Run all tests |
| `make load-test` | Execute load testing |
| `make clean` | Clean generated binaries |

## 🏗️ Project Architecture

### Directory Structure

- `cmd/` - Application entry points
  - `api/` - Main application server
  - `loadtest/` - Load testing utilities
  - `setup/` - Setup utilities (HTTP server, middleware, worker)
- `internal/` - Private application code
  - `backoffice/` - Consumer registration logic
  - `cfg/` - Configuration management
  - `domain/` - Domain models and business logic
  - `eventqueue/` - Event processing and queue handling
  - `interstore/` - Data storage interfaces
- `pkg/` - Public packages that can be imported
  - `asynqsvc/` - Async queue service utilities
  - `cache/` - Caching layer
  - `httpsvc/` - HTTP service utilities
  - `logs/` - Logging utilities
- `example/` - Example implementations
- `docs/` - Documentation and assets

### Key Components

- **Event Queue**: Handles event processing using Asynq
- **HTTP API**: REST API for webhook management
- **Worker**: Background job processor
- **Cache**: Redis-based caching strategy
- **Storage**: MongoDB for data persistence

## 🧪 Testing

### Running Tests

```bash
# Run all tests
make test

# Run tests with verbose output
go test -v ./...

# Run tests for specific package
go test -v ./internal/eventqueue

# Run tests with coverage
go test -v -cover ./...
```

### Test Structure

- Unit tests are located alongside the source code with `_test.go` suffix
- Integration tests use testcontainers for external dependencies
- Load tests are available in `cmd/loadtest/`

### Writing Tests

- Follow Go testing conventions
- Use table-driven tests where appropriate
- Mock external dependencies using the provided mock interfaces
- Include both positive and negative test cases

## 📝 Code Style and Guidelines

### Go Code Standards

- Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `go fmt` to format your code
- Use `go vet` to check for common errors
- Follow Go naming conventions
- Write clear, descriptive comments for exported functions and types

### Code Organization

- Keep functions small and focused
- Use interfaces to define contracts
- Separate business logic from infrastructure concerns
- Follow the Clean Architecture principles evident in the project structure

### Error Handling

- Use Go's idiomatic error handling
- Wrap errors with context using `fmt.Errorf`
- Define custom error types in the `domain` package when appropriate
- Log errors at appropriate levels

## 🐛 Bug Reports and Feature Requests

### Bug Reports

When filing a bug report, please include:

1. **Description**: Clear description of the issue
2. **Steps to reproduce**: Minimal steps to reproduce the behavior
3. **Expected behavior**: What you expected to happen
4. **Actual behavior**: What actually happened
5. **Environment**: Go version, OS, Docker version
6. **Logs**: Relevant log output or error messages

### Feature Requests

When proposing a feature:

1. **Use case**: Explain the problem this feature solves
2. **Proposed solution**: Describe your proposed approach
3. **Alternatives**: Any alternative solutions you considered
4. **Impact**: How this affects existing functionality

## 🔄 Pull Request Process

### Before Submitting

1. Ensure your code follows the project's style guidelines
2. Add tests for new functionality
3. Update documentation if needed
4. Ensure all tests pass
5. Update the README if you're adding new features

### Submission Process

1. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes**
   - Write clear, concise commit messages
   - Keep commits focused and atomic
   - Follow conventional commit format: `type: description`

3. **Test your changes**
   ```bash
   make test
   go vet ./...
   go fmt ./...
   ```

4. **Push and create a PR**
   ```bash
   git push origin feature/your-feature-name
   ```

5. **PR Requirements**
   - Provide clear description of changes
   - Reference any related issues
   - Ensure CI passes
   - Request review from maintainers

### Commit Message Format

We follow conventional commits:

```
<type>: <description>

[optional body]

[optional footer]
```

Types:
- `feat`: New features
- `fix`: Bug fixes
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

## 🐳 Docker and Deployment

### Local Development with Docker

```bash
# Start all services including dependencies
docker compose up

# Start only dependencies
docker compose up -d redis mongodb

# Build and run the application container
docker build -t gqueue .
docker run -p 8080:8080 gqueue
```

### Docker Image

The project provides a pre-built Docker image: `isaacdsc/gqueue:latest`

## 📚 Documentation

- Keep README.md up to date
- Document new features and APIs
- Update Docker documentation when changing container setup
- Add examples for new functionality

## 🤝 Community Guidelines

- Be respectful and inclusive
- Provide constructive feedback
- Help others learn and grow
- Follow the project's code of conduct

## 📄 License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.

## 💬 Getting Help

- Check existing issues and documentation first
- Create an issue for questions or discussion
- Provide context and details when asking for help

## 🎯 Good First Issues

Look for issues labeled with:
- `good first issue`
- `help wanted`
- `documentation`

These are great starting points for new contributors!

---

Thank you for contributing to IWebhook! 🎉
