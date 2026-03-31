# ─────────────────────────────────────────────────────────────────────────────
# Stage 1 – cross-compile Linux → Windows amd64
# ─────────────────────────────────────────────────────────────────────────────
FROM golang:1.22-bullseye AS builder

# MinGW cross-compiler required by Fyne (CGO)
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc-mingw-w64-x86-64 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY . .

# Generate go.sum and download dependencies
RUN go mod tidy && go mod download

RUN CGO_ENABLED=1 \
    GOOS=windows \
    GOARCH=amd64 \
    CC=x86_64-w64-mingw32-gcc \
    go build \
      -ldflags="-H windowsgui -s -w" \
      -o wsl_proxy.exe \
      .

# ─────────────────────────────────────────────────────────────────────────────
# Stage 2 – export artefact only (scratch image)
# ─────────────────────────────────────────────────────────────────────────────
FROM scratch AS artifact
COPY --from=builder /app/wsl_proxy.exe /wsl_proxy.exe
