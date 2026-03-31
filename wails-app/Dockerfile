# ─── Stage 1: builder ────────────────────────────────────────────────────────
FROM golang:1.22-bullseye AS builder

# System dependencies:
#   gcc                   — native Linux gcc for Wails bindings generation (host step)
#   gcc-mingw-w64-x86-64  — Windows cross-compiler; Go auto-detects it for windows/amd64
#   curl / ca-certificates — download Node.js setup script
#   pkg-config            — required by some Wails internal deps
RUN apt-get update && apt-get install -y --no-install-recommends \
        gcc \
        gcc-mingw-w64-x86-64 \
        curl \
        ca-certificates \
        pkg-config \
    && curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*

# Install Wails CLI
RUN go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0

WORKDIR /build

# Copy go module files first for better layer caching
COPY go.mod go.sum ./
RUN go mod tidy

# Copy the rest of the source
COPY . .

# Install frontend dependencies
RUN cd frontend && npm ci

# Cross-compile to Windows amd64.
# - Do NOT set GOOS/GOARCH/CC globally: Wails first compiles a host (Linux) binary
#   for bindings generation using the native gcc; setting CC to MinGW here would
#   break that step (MinGW doesn't have Linux headers like sigset_t).
# - CGO_ENABLED=1 is needed so Wails can build.
# - When Wails internally sets GOOS=windows GOARCH=amd64, the Go toolchain
#   automatically uses x86_64-w64-mingw32-gcc from PATH as the C compiler.
RUN CGO_ENABLED=1 \
    CGO_LDFLAGS="-static-libgcc -static-libstdc++" \
    wails build \
        -platform windows/amd64 \
        -devtools \
        -ldflags "-s -w" \
        -o wsl_proxy.exe

# Copy artifact to /output so the host can extract it via a volume mount
RUN mkdir -p /output && cp build/bin/wsl_proxy.exe /output/wsl_proxy.exe

# ─── Stage 2: export artifact only ───────────────────────────────────────────
FROM scratch AS artifact
COPY --from=builder /output/wsl_proxy.exe /wsl_proxy.exe
