# ── Build Stage ───────────────────────────────────
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Download dependencies first (cached layer)
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /server \
    ./cmd/server

# ── Run Stage ─────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates wget

# Non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=builder /server /server
COPY --from=builder /app/db/migrations /migrations

USER appuser

EXPOSE 8080

ENTRYPOINT ["/server"]
