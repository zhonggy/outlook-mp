# outlook-manager Docker 多阶段构建
# 前端 → 后端 → 最小运行镜像

# ---- 阶段 1：构建前端 ----
FROM node:22-alpine AS frontend
WORKDIR /build/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ .
RUN npm run build

# ---- 阶段 2：构建后端 ----
FROM golang:1.25-alpine AS backend
RUN apk add --no-cache git
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /build/web/dist web/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/outlook-manager ./cmd/server

# ---- 阶段 3：运行镜像 ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -h /app outlook
WORKDIR /app
COPY --from=backend /build/bin/outlook-manager .
USER outlook
EXPOSE 18327
VOLUME ["/app/data", "/app/configs"]
ENTRYPOINT ["./outlook-manager"]