# =============================================================================
# Stage 1: Build
# Using golang:1.23-alpine as the build image — Alpine is smaller than the
# full Debian-based image, and the CGO_ENABLED=0 flag produces a statically
# linked binary that doesn't need libc at runtime.
# =============================================================================
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install git (required for go mod download with some VCS dependencies)
RUN apk add --no-cache git

# Cache dependencies separately from source code.
# Docker layer caching means `go mod download` only reruns when go.mod/go.sum change,
# not on every source code change — significantly faster rebuilds.
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .

# CGO_ENABLED=0: static binary (no C runtime dependency)
# GOOS=linux: cross-compile targeting Linux container runtime
# -ldflags "-s -w": strip debug symbols for smaller binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /taskflow \
    ./cmd/api

# =============================================================================
# Stage 2: Runtime
# gcr.io/distroless/static:nonroot — no shell, no package manager, no libc.
# Only contains CA certificates and timezone data. Significantly reduces the
# attack surface compared to Alpine or Debian runtime images.
# The statically linked Go binary runs without any OS dependencies.
# =============================================================================
FROM gcr.io/distroless/static:nonroot

WORKDIR /

# Copy the compiled binary from the build stage
COPY --from=builder /taskflow /taskflow

# Note: migrations and seeds are embedded inside the binary via //go:embed
# so no additional COPY steps are needed for those files.

# Run as non-root user (uid=65532, gid=65532 in distroless:nonroot)
USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/taskflow"]
