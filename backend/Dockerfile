# Multi-stage build for Go backend
FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

# Copy dependency manifests FIRST for optimal Docker layer caching.
# Dependencies are only re-downloaded when go.mod or go.sum change.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build a statically linked binary with stripped debug info
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/taskmgr-api ./cmd/api

# Final minimal production image
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/bin/taskmgr-api /app/taskmgr-api
COPY --from=builder /app/migrations /app/migrations

RUN mkdir -p /app/uploads && chown -R appuser:appgroup /app /app/uploads && chmod 750 /app/uploads

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/taskmgr-api"]
