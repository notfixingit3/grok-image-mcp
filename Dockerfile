# Stage 1: Build the statically linked Go binary
# Locally confirmed toolchain: Go 1.27 (go1.27.4)
FROM golang:1.27-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY *.go ./
COPY assets/sample_output.png ./assets/sample_output.png

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o grok-image-mcp .

# Stage 2: Run in a minimal alpine container
# TODO: Verify latest upstream Alpine tag on Docker Hub when building in CI
FROM alpine:3.21
RUN apk --no-cache add ca-certificates

COPY --from=builder /app/grok-image-mcp /usr/local/bin/grok-image-mcp

ENTRYPOINT ["/usr/local/bin/grok-image-mcp"]