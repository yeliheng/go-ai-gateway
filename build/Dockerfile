# Build Stage
FROM golang:alpine AS builder

# Set environment variables
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Install git for fetching dependencies
RUN apk add --no-cache git

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build services
# We build all binaries in this stage to maximize layer caching
RUN go build -o /bin/gateway ./cmd/gateway/main.go && \
    go build -o /bin/identity ./cmd/identity/main.go && \
    go build -o /bin/agent ./cmd/agent/main.go && \
    go build -o /bin/biz ./cmd/biz/main.go

# Shared runtime configuration
FROM alpine:latest AS runtime_base
RUN apk --no-cache add ca-certificates tzdata curl
WORKDIR /app

# Identity Service Image
FROM runtime_base AS identity
COPY --from=builder /bin/identity /app/identity
COPY --from=builder /app/config /app/config
EXPOSE 50051
ENTRYPOINT ["/app/identity"]

# Agent Service Image
FROM runtime_base AS agent
COPY --from=builder /bin/agent /app/agent
COPY --from=builder /app/config /app/config
EXPOSE 50052
ENTRYPOINT ["/app/agent"]

# Gateway Service Image
FROM runtime_base AS gateway
COPY --from=builder /bin/gateway /app/gateway
COPY --from=builder /app/config /app/config
EXPOSE 8080
ENTRYPOINT ["/app/gateway"]

# Biz Service Image
FROM runtime_base AS biz
COPY --from=builder /bin/biz /app/biz
COPY --from=builder /app/config /app/config
EXPOSE 8081
ENTRYPOINT ["/app/biz"]
