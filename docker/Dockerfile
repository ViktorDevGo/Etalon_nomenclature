# Build stage
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /build/app \
    ./cmd/app

# Final stage - using debian for LibreOffice
FROM debian:bookworm-slim

# Install LibreOffice and dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    libreoffice-calc \
    libreoffice-writer \
    ca-certificates \
    tzdata && \
    rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN groupadd -g 1000 appuser && \
    useradd -m -u 1000 -g appuser appuser

WORKDIR /app

# Create temp directory for XLS conversion
RUN mkdir -p /tmp/xls-convert && \
    chown -R appuser:appuser /tmp/xls-convert

# Copy binary from builder
COPY --from=builder /build/app /app/app

# Copy entrypoint script
COPY docker/entrypoint.sh /app/entrypoint.sh

# Make entrypoint executable and change ownership
RUN chmod +x /app/entrypoint.sh && \
    chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port (if needed for health checks)
EXPOSE 8080

# Set default config path
ENV CONFIG_PATH=/app/config.yaml

# Set LibreOffice to run in headless mode
ENV HOME=/tmp

# Set entrypoint
ENTRYPOINT ["/app/entrypoint.sh"]

# Run the application
CMD ["/app/app"]
