# Build stage
FROM golang:1.23-alpine AS builder

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

# Build the binary for target platform
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -a -installsuffix cgo -ldflags="-s -w" -o harness-onboarder .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 onboarder && \
    adduser -D -s /bin/sh -u 1000 -G onboarder onboarder

# Set working directory
WORKDIR /app

# Copy binary and entrypoint from builder stage
COPY --from=builder /app/harness-onboarder .
COPY entrypoint.sh .

# Make entrypoint executable and change ownership to non-root user
RUN chmod +x /app/entrypoint.sh && \
    chown onboarder:onboarder /app/harness-onboarder /app/entrypoint.sh

# Switch to non-root user
USER onboarder

# Set entrypoint
ENTRYPOINT ["./entrypoint.sh"]

# Default command
CMD ["./harness-onboarder", "--help"]