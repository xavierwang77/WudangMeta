# 第一阶段：构建阶段
FROM golang:1.24 AS builder

# 设置工作目录
WORKDIR /app

# 将 go.mod 和 go.sum 文件复制到工作目录
COPY go.mod go.sum ./

# 设置 Go Modules 镜像源为 goproxy.io
ENV GOPROXY=https://goproxy.io,direct

# 执行 go mod tidy 清理和更新依赖
RUN go mod tidy

# 下载依赖模块
RUN go mod download

# 将项目的所有文件复制到工作目录
COPY . .

# 设置版本变量（通过 --build-arg 注入）
ARG VERSION=dev

# 获取依赖并编译可执行文件，注入版本号
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags "-X 'WudangMeta/cmn.Version=${VERSION}'" \
    -o main .

# 第二阶段：运行阶段
FROM alpine:latest

# 切换为阿里云的镜像源，并安装必要的库
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
    apk update && \
    apk --no-cache add ca-certificates curl bash

# 设置工作目录
WORKDIR /app

# 从构建阶段复制编译好的可执行文件和本地配置文件
COPY --from=builder /app/main .

# 设置时区
ENV TIME_ZONE=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TIME_ZONE /etc/localtime && echo $TIME_ZONE > /etc/timezone

# 暴露必要的端口（如有需要）
EXPOSE 3388

# 设置容器启动时执行的命令
ENTRYPOINT ["/app/main", "serve"]
