# NeoAgent Dockerfile
# @author: sun977
# @date: 2025.10.21
# @description: Agent端Docker镜像构建文件

# 构建阶段
FROM golang:1.22-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装必要的工具和依赖
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    gcc \
    musl-dev

# 复制go mod文件并下载依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用程序
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=$(date +%Y%m%d-%H%M%S) -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o neoAgent \
    cmd/agent/main.go

# 运行阶段
FROM alpine:latest

# 安装运行时依赖和安全工具
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    wget \
    nmap \
    nmap-scripts \
    masscan \
    && rm -rf /var/cache/apk/*

# 安装Nuclei
RUN wget -q https://github.com/projectdiscovery/nuclei/releases/latest/download/nuclei_3.2.0_linux_amd64.zip \
    && unzip nuclei_3.2.0_linux_amd64.zip \
    && mv nuclei /usr/local/bin/ \
    && rm nuclei_3.2.0_linux_amd64.zip \
    && chmod +x /usr/local/bin/nuclei

# 下载Nuclei模板
RUN nuclei -update-templates

# 创建非root用户
RUN addgroup -g 1001 neoagent \
    && adduser -D -u 1001 -G neoagent neoagent

# 设置工作目录
WORKDIR /app

# 复制构建的二进制文件
COPY --from=builder /app/neoAgent .

# 复制配置文件
COPY --from=builder /app/configs ./configs

# 创建必要的目录
RUN mkdir -p \
    /app/work \
    /app/temp \
    /app/logs \
    /app/data \
    /app/nuclei-templates \
    && chown -R neoagent:neoagent /app

# 设置时区
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# 设置环境变量
ENV ENVIRONMENT=production
ENV SERVER_PORT=8081
ENV LOG_LEVEL=info
ENV LOG_OUTPUT=file
ENV AGENT_WORK_DIR=/app/work
ENV AGENT_TEMP_DIR=/app/temp
ENV AGENT_LOG_DIR=/app/logs
ENV AGENT_DATA_DIR=/app/data
ENV NUCLEI_TEMPLATES_DIR=/app/nuclei-templates
ENV NMAP_PATH=/usr/bin/nmap
ENV NUCLEI_PATH=/usr/local/bin/nuclei
ENV MASSCAN_PATH=/usr/bin/masscan

# 切换到非root用户
USER neoagent

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8081/health || exit 1

# 暴露端口
EXPOSE 8081

# 设置启动命令
CMD ["./neoAgent"]