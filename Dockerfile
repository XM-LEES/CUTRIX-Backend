# 开发环境 Dockerfile
FROM golang:1.24-alpine AS development

# 安装必要的工具
RUN apk add --no-cache git

# 设置工作目录
WORKDIR /app

# 复制 go mod 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 暴露端口
EXPOSE 8080

# 启动开发服务器（支持热重载，需要安装 air 或使用 go run）
CMD ["go", "run", "./cmd/api"]

