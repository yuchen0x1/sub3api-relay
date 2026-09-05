FROM golang:1.26.6-alpine

WORKDIR /app

# 安装必要的工具
RUN apk add --no-cache git

# 复制go.mod和go.sum
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN VERSION_VALUE="$(./scripts/resolve-version.sh)" && \
    go build -ldflags="-s -w -X main.Version=${VERSION_VALUE} -X main.BuildType=docker" -o main ./cmd/server/

# 运行时目录（数据/日志）预建并授予非 root 用户
RUN addgroup -g 1000 sub3api && \
    adduser -D -u 1000 -G sub3api sub3api && \
    mkdir -p /app/data /app/logs && \
    chown -R sub3api:sub3api /app

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/health || exit 1

# 以非 root 身份运行
USER sub3api

# 运行应用
CMD ["./main"]
