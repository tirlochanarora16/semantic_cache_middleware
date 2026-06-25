# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy and download dependencies first (for layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/server .

# Runtime stage
FROM alpine:3.20

WORKDIR /app

# Install ca-certificates for HTTPS calls to OpenAI
RUN apk add --no-cache ca-certificates

# Copy the binary from builder
COPY --from=builder /bin/server /app/server

# Expose the port
EXPOSE 3000

# Run the server
ENTRYPOINT ["/app/server"]