# Stage 1: Build (no CGO, static binary)
FROM golang:1.23.4-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app ./cmd/server

# Stage 2: Production image with headless Chromium for PDF rendering
FROM chromedp/headless-shell:131.0.6778.86
RUN groupadd --system -g 10001 appuser && useradd --system -u 10001 -g appuser --no-create-home --shell /sbin/nologin appuser
COPY --chown=10001:10001 --from=builder /app /app
ENV CHROMIUM_PATH=/headless-shell/headless-shell
ENV GIN_MODE=release
EXPOSE 8080
USER appuser
ENTRYPOINT ["/app"]
