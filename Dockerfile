# Stage 1: Build the statically linked Go binary
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY main.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o grok-image-mcp main.go

# Stage 2: Run in a minimal alpine container
FROM alpine:3.19
RUN apk --no-cache add ca-certificates

COPY --from=builder /app/grok-image-mcp /usr/local/bin/grok-image-mcp

ENTRYPOINT ["/usr/local/bin/grok-image-mcp"]