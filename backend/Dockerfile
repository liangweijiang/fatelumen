# Development image: source is bind-mounted by the root compose file.
# It intentionally includes Chromium and CJK fonts so PDF development behaves
# like the runtime image.
FROM golang:1.26-bookworm AS development
RUN apt-get update \
    && apt-get install -y --no-install-recommends chromium fonts-noto-cjk ca-certificates \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /src
ENV CHROMIUM_PATH=/usr/bin/chromium

# Build a static production binary.
FROM golang:1.26-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app ./cmd/server

# Production image with a supported Chromium package and CJK fonts.
FROM debian:bookworm-slim AS production
RUN apt-get update \
    && apt-get install -y --no-install-recommends chromium fonts-noto-cjk ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --system -g 10001 appuser \
    && useradd --system -u 10001 -g appuser --no-create-home --shell /usr/sbin/nologin appuser
COPY --chown=10001:10001 --from=builder /app /app
ENV CHROMIUM_PATH=/usr/bin/chromium
ENV GIN_MODE=release
EXPOSE 8080
USER appuser
ENTRYPOINT ["/app"]
