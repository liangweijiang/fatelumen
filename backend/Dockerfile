# Stage 1: Build (no CGO, static binary)
FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app ./cmd/server

# Stage 2: Production image with headless Chromium for PDF rendering
FROM chromedp/headless-shell:latest
COPY --from=builder /app /app
ENV CHROMIUM_PATH=/headless-shell/headless-shell
ENV GIN_MODE=release
EXPOSE 8080
ENTRYPOINT ["/app"]
