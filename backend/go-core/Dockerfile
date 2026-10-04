# syntax=docker/dockerfile:1

# ---- Builder stage: compile a static binary ----
FROM golang:1.23-alpine AS builder
WORKDIR /src

# Cache module downloads separately from source.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# CGO_ENABLED=0 → fully static binary; -ldflags keeps the image lean.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/api ./cmd/api

# ---- Runtime stage: minimal, non-root, read-only ----
FROM alpine:3.19
RUN apk add --no-cache ca-certificates \
    && addgroup -S -g 10001 app \
    && adduser -S -D -H -u 10001 -G app app

WORKDIR /app

COPY --from=builder /out/api /app/api

# The API writes nothing to disk; keep the filesystem read-only.
USER 10001:10001
EXPOSE 8081 9090

# OCI healthcheck — used by `docker run`/docker-compose (ignored by K8s,
# which uses the pod-level probes).
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:8081/health || exit 1

ENTRYPOINT ["/app/api"]
