# ============================================================================
# MiniGaraj Scraper - Dockerfile
# Multi-stage build: builder + minimal runtime
# ============================================================================

# Stage 1: Build
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install dependencies for CGO (needed by some libs)
RUN apk add --no-cache gcc musl-dev

# Download modules first (cache layer)
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o /app/scraper ./cmd/scraper

# ============================================================================
# Stage 2: Runtime
# ============================================================================
FROM alpine:3.19

WORKDIR /app

# CA certs for HTTPS scraping
RUN apk add --no-cache ca-certificates tzdata

# Copy binary and migrations
COPY --from=builder /app/scraper ./scraper
COPY --from=builder /app/migrations ./migrations

# Non-root user
RUN addgroup -S scraper && adduser -S scraper -G scraper
USER scraper

EXPOSE 8300

ENTRYPOINT ["./scraper"]
