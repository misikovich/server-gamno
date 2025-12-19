# --- BUILD STAGE ---
# Using golang:1.25.5-bookworm to match your Go version and target Debian 12
FROM golang:1.25.5-bookworm AS builder

# Set working directory
WORKDIR /app

# Install build dependencies for CGO (required by github.com/mattn/go-sqlite3)
# Debian 12 (Bookworm) uses GLIBC 2.36, which natively supports symbols from 2.32, 2.33, and 2.34
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    libc6-dev \
    pkg-config \
    && rm -rf /var/lib/apt/lists/*

# Copy dependency files first
COPY go.mod go.sum* ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the binary with CGO enabled
# We build specifically in this environment to ensure it links against the correct GLIBC
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o server .

# --- RUN STAGE ---
# Debian 12 (Bookworm) is the requested target server OS
FROM debian:bookworm-slim

WORKDIR /app

# Install runtime dependencies (ca-certificates for HTTPS/YouTube API, sqlite3 for DB)
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    sqlite3 \
    && rm -rf /var/lib/apt/lists/*

# Copy the binary from the build stage
COPY --from=builder /app/server .

# Copy environment file and data files
# NOTE: Ensure your .env file is configured for the Docker environment (e.g., HOST=0.0.0.0)
COPY .env .
COPY videos.json .

# Note: We don't 'touch' or use 'VOLUME' here because we will 
# bind-mount the file from the host at runtime using '-v'.
# This avoids the "cannot mount volume over existing file" error.

# Expose the server port
EXPOSE 1488

# Important: HOST must be 0.0.0.0 inside Docker to allow external access.
# api.femboyz.cloud is your external address; internal binding must be 0.0.0.0.
ENV HOST=0.0.0.0
ENV PORT=1488

# Run the server
CMD ["./server"]