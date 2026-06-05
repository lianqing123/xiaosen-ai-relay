FROM golang:1.26.2-alpine

WORKDIR /app

# 安装运行期需要的工具：
# - bash/python3/curl/docker-cli 给 Kiro 激活脚本使用
# - unzip/wget/sha256sum 给 Kiro CLI 安装与健康检查使用
RUN apk add --no-cache bash curl docker-cli git python3 unzip wget coreutils

# 复制go.mod和go.sum
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN go build -o main ./cmd/server/

# 暴露端口
EXPOSE 8080

# 运行应用
CMD ["./main"]
