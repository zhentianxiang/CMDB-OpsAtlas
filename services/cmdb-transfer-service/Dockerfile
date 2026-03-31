FROM golang:1.22-alpine AS builder
ENV GO111MODULE=on GOPROXY=https://goproxy.cn,direct
WORKDIR /app
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o cmdb-transfer-service services/cmdb-transfer-service/cmd/main.go

FROM mysql:8.0
RUN microdnf install -y tzdata curl ca-certificates && microdnf clean all
WORKDIR /app
COPY --from=builder /app/cmdb-transfer-service .
EXPOSE 8080
CMD ["./cmdb-transfer-service"]
