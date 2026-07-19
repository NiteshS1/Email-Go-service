# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy dependency files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/email-service ./cmd/main.go

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/email-service .

# Copy runtime assets (migrations and templates must be beside binary for app to find them)
COPY --from=builder /build/migrations ./migrations
COPY --from=builder /build/templates ./templates

EXPOSE 8080

# Config via env (DB_*, RABBITMQ_*, APP_PORT, etc.)
ENTRYPOINT ["./email-service"]
