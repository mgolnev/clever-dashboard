# syntax=docker/dockerfile:1

# --- Stage 1: сборка фронтенда (Vite -> dist) ---
FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci || npm install
COPY frontend/ ./
RUN npm run build

# --- Stage 2: сборка бэкенда (статический Go-бинарь, без CGO) ---
FROM golang:1.26-alpine AS backend
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# --- Stage 3: рантайм ---
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget
WORKDIR /app
COPY --from=backend /out/server /app/server
COPY --from=frontend /app/frontend/dist /app/web
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

# /data — постоянное хранилище (volume в Docker, persistenceMount в Amvera).
# PORT=80: Amvera по умолчанию маршрутизирует входной трафик на порт 80
# контейнера; процесс запускается от root (штатный паттерн Amvera), поэтому
# bind на привилегированный порт доступен без setcap.
ENV PORT=80 \
    DB_DRIVER=sqlite \
    DB_DSN=/data/clever.db \
    STATIC_DIR=/app/web

RUN chmod +x /usr/local/bin/docker-entrypoint.sh && mkdir -p /data
VOLUME ["/data"]
EXPOSE 80

HEALTHCHECK --interval=30s --timeout=3s --start-period=15s --retries=3 \
    CMD wget -qO- "http://127.0.0.1:${PORT}/api/health" || exit 1

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
