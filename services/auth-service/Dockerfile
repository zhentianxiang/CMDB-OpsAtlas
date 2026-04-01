# 构建阶段
FROM golang:1.22-alpine AS builder

# 启用 Go Modules + 配置国内七牛云代理（最快）
ENV GO111MODULE=on GOPROXY=https://goproxy.cn,direct

WORKDIR /app

# 复制 go.mod 和 go.sum (如果有)
# 注意：在 Monorepo 中，我们可能需要复制根目录的 go.work 和所有相关的 go.mod
COPY . .

# 自动补全 go.sum 并下载依赖
RUN go mod tidy

# 编译 auth-service
RUN CGO_ENABLED=0 go build -o auth-service services/auth-service/cmd/main.go

# 运行阶段
FROM alpine:latest
RUN sed -i 's|https://dl-cdn.alpinelinux.org/alpine|https://mirrors.aliyun.com/alpine|g' /etc/apk/repositories \
    && apk add --no-cache curl ca-certificates tzdata \
    && update-ca-certificates \
    && ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone
ENV TZ=Asia/Shanghai

WORKDIR /app

COPY --from=builder /app/auth-service .

EXPOSE 8080

CMD ["./auth-service"]
