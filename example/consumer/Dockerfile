# syntax=docker/dockerfile:1

# Build stage
FROM golang:1.24.3 AS build
WORKDIR /src

# Initialize go module for the consumer app
RUN go mod init consumer

# Copy the source code
COPY main.go .

# Build the application
RUN CGO_ENABLED=0 go build -o /bin/consumer ./main.go

# Runtime stage
FROM alpine:latest AS final

# Install runtime dependencies
RUN apk --update add ca-certificates tzdata && \
    update-ca-certificates

# Create non-privileged user
ARG UID=10001
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    appuser
USER appuser

# Copy executable
COPY --from=build /bin/consumer /bin/

# Expose port
EXPOSE 3333

# Run the application
ENTRYPOINT [ "/bin/consumer" ]
