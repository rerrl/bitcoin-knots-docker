# Multi-stage build: compile the tiny stdlib-only Go server, then run it in a
# slim runtime. No build-time network beyond fetching the pinned Go module
# cache (the module has zero external dependencies, so nothing to download for
# the program itself).

FROM golang:1.24-bookworm AS builder

WORKDIR /src

# Copy only what's needed for the build (go.mod with no deps + sources).
COPY go.mod ./
COPY main.go ./
COPY web ./web

ENV CGO_ENABLED=0 GOOS=linux
# -trimpath keeps the binary reproducible; -ldflags optimizes size.
RUN go build -trimpath -ldflags="-s -w" -o dashboard .

# ---------------------------------------------------------------------------
# Runtime stage
# ---------------------------------------------------------------------------
FROM debian:bookworm-slim

# No runtime .so dependencies for a static Go binary; base image is enough.

# Get host user ID so the dashboard can read bitcoind's .cookie and logs.
# bitcoind runs as ${USER_ID}:${GROUP_ID} so matching here means the mounted
# datadir files are readable without running as root.
ARG USER_ID=1000
ARG GROUP_ID=1000

RUN groupadd -g ${GROUP_ID} dashboard && \
    useradd -u ${USER_ID} -g dashboard -m dashboard

COPY --from=builder /src/dashboard /usr/local/bin/dashboard

# Runtime settings come from env (set in docker-compose.yaml); the baked-in
# defaults already match the default compose mounts.

EXPOSE 8080

USER dashboard
WORKDIR /home/dashboard

ENTRYPOINT ["/usr/local/bin/dashboard"]