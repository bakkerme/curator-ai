# Stage 1: Build Go backend
FROM golang:1.24-alpine AS backend-builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY backend/ ./backend/
COPY cmd/ ./cmd/
COPY --from=type-generator /app/shared/ ./shared/

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/curator ./cmd/curator

# Stage 2: Final runtime image
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata wget

WORKDIR /app

# Copy built application
COPY --from=backend-builder /app/bin/curator ./bin/curator

# Copy configuration templates
COPY configs/ ./configs/

# Create necessary directories
RUN mkdir -p data logs

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/v1/health || exit 1

# Run as non-root user
RUN adduser -D -s /bin/sh curator
RUN chown -R curator:curator /app
USER curator

CMD ["./bin/curator"]