# Build stage
FROM golang:1.21-alpine AS builder

# Install git and ca-certificates for private repositories and HTTPS
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-s -w" -o harness-onboarder .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 onboarder && \
    adduser -D -s /bin/sh -u 1000 -G onboarder onboarder

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/harness-onboarder .

# Change ownership to non-root user
RUN chown onboarder:onboarder /app/harness-onboarder

# Switch to non-root user
USER onboarder

# Set entrypoint
ENTRYPOINT ["./harness-onboarder"]

# Default command
CMD ["--help"]