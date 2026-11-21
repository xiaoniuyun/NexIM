# ---------- 阶段1：构建 Admin 前端 ----------
FROM node:20-alpine AS admin-builder
WORKDIR /web-admin
COPY web-admin/package*.json ./
RUN npm install --no-audit --no-fund
COPY web-admin/ .
RUN npm run build

# ---------- 阶段2：构建 Go 后端（嵌入前端） ----------
FROM golang:1.23-alpine AS builder
ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /src
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ .
# 将前端产物放入 webassets/dist 供 go:embed
RUN mkdir -p webassets/dist/admin
COPY --from=admin-builder /web-admin/dist/ ./webassets/dist/admin/
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/nexim-server ./cmd/server

# ---------- 阶段3：运行（单二进制，API + WS + Admin） ----------
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata ffmpeg \
    && addgroup -S nexim && adduser -S -G nexim nexim
WORKDIR /app
COPY --from=builder /out/nexim-server ./nexim-server
# 同时提供 lite / standard 两套配置，运行时用 CONFIG_PATH 选择
COPY server/configs/ ./configs/
RUN chown -R nexim:nexim /app
USER nexim
EXPOSE 8080
ENV CONFIG_PATH=/app/configs/config.yaml
ENTRYPOINT ["./nexim-server"]
