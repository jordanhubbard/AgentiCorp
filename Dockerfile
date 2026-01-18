# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod ./
COPY go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o arbiter \
    ./cmd/arbiter

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 arbiter && \
    adduser -D -u 1000 -G arbiter arbiter

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/arbiter /app/arbiter

# Change ownership
RUN chown -R arbiter:arbiter /app

# Switch to non-root user
USER arbiter

# Expose port (if needed in future)
EXPOSE 8080

# Set entrypoint
ENTRYPOINT ["/app/arbiter"]
