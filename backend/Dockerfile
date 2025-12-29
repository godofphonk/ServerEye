# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy entire project (needed for replace directives)
COPY . .

# Change to backend directory for build
WORKDIR /app/backend

# Download dependencies
RUN go mod download

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/servereye-backend .

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata curl

# Create non-root user
RUN addgroup -g 1001 -S servereye && \
    adduser -u 1001 -S servereye -G servereye

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/servereye-backend .

# Create logs directory
RUN mkdir -p /app/logs && chown -R servereye:servereye /app

# Switch to non-root user
USER servereye

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Expose port
EXPOSE 8080

# Run binary
CMD ["./servereye-backend"]
