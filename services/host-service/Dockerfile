FROM golang:1.22-alpine AS builder
ENV GO111MODULE=on GOPROXY=https://goproxy.cn,direct
WORKDIR /app
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o host-service services/host-service/cmd/main.go

FROM alpine:latest
RUN sed -i 's|https://dl-cdn.alpinelinux.org/alpine|https://mirrors.aliyun.com/alpine|g' /etc/apk/repositories \
    && apk add --no-cache curl ca-certificates \
    && update-ca-certificates
WORKDIR /app
COPY --from=builder /app/host-service .
EXPOSE 8080
CMD ["./host-service"]
