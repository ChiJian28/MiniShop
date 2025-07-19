# 使用官方 Go 镜像作为构建环境
FROM golang:1.21-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装必要的包
RUN apk add --no-cache git ca-certificates tzdata

# 复制 go mod 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/main.go

# 使用 scratch 作为最终镜像
FROM alpine:latest

# 安装 ca-certificates 用于 HTTPS 请求
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# 从构建阶段复制二进制文件
COPY --from=builder /app/main .

# 复制配置文件
COPY --from=builder /app/config ./config

# 暴露端口
EXPOSE 8082

# 设置时区
ENV TZ=Asia/Shanghai

# 运行应用
CMD ["./main"] 