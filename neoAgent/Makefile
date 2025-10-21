# NeoAgent Makefile
# @author: sun977
# @date: 2025.10.21
# @description: Agent端构建和部署脚本

# 变量定义
APP_NAME := neoAgent
VERSION := $(shell date +%Y%m%d-%H%M%S)
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GO_VERSION := $(shell go version | awk '{print $$3}')

# 构建参数
LDFLAGS := -ldflags="-w -s -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)"
BUILD_DIR := bin
DOCKER_IMAGE := neoscan/agent
DOCKER_TAG := latest

# Go相关变量
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := $(GOCMD) fmt
GOVET := $(GOCMD) vet

# 平台相关
BINARY_NAME := $(APP_NAME)
BINARY_UNIX := $(BINARY_NAME)_unix
BINARY_WINDOWS := $(BINARY_NAME).exe
BINARY_DARWIN := $(BINARY_NAME)_darwin

# 默认目标
.PHONY: all
all: clean deps fmt vet test build

# 帮助信息
.PHONY: help
help:
	@echo "NeoAgent构建脚本"
	@echo ""
	@echo "可用命令:"
	@echo "  build          构建应用程序"
	@echo "  build-all      构建所有平台版本"
	@echo "  build-linux    构建Linux版本"
	@echo "  build-windows  构建Windows版本"
	@echo "  build-darwin   构建macOS版本"
	@echo "  clean          清理构建文件"
	@echo "  deps           下载依赖"
	@echo "  fmt            格式化代码"
	@echo "  vet            代码静态检查"
	@echo "  test           运行测试"
	@echo "  test-coverage  运行测试并生成覆盖率报告"
	@echo "  lint           代码质量检查"
	@echo "  run            运行应用程序"
	@echo "  dev            开发模式运行"
	@echo "  docker-build   构建Docker镜像"
	@echo "  docker-run     运行Docker容器"
	@echo "  docker-dev     开发环境Docker运行"
	@echo "  docker-push    推送Docker镜像"
	@echo "  install        安装到系统"
	@echo "  uninstall      从系统卸载"
	@echo "  version        显示版本信息"

# 构建
.PHONY: build
build: deps
	@echo "构建 $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) cmd/agent/main.go
	@echo "构建完成: $(BUILD_DIR)/$(BINARY_NAME)"

# 构建所有平台版本
.PHONY: build-all
build-all: build-linux build-windows build-darwin

# 构建Linux版本
.PHONY: build-linux
build-linux: deps
	@echo "构建Linux版本..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_UNIX) cmd/agent/main.go
	@echo "Linux版本构建完成: $(BUILD_DIR)/$(BINARY_UNIX)"

# 构建Windows版本
.PHONY: build-windows
build-windows: deps
	@echo "构建Windows版本..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_WINDOWS) cmd/agent/main.go
	@echo "Windows版本构建完成: $(BUILD_DIR)/$(BINARY_WINDOWS)"

# 构建macOS版本
.PHONY: build-darwin
build-darwin: deps
	@echo "构建macOS版本..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_DARWIN) cmd/agent/main.go
	@echo "macOS版本构建完成: $(BUILD_DIR)/$(BINARY_DARWIN)"

# 清理
.PHONY: clean
clean:
	@echo "清理构建文件..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -rf tmp
	rm -rf logs/*.log
	rm -rf data/*.db
	rm -rf work/*
	rm -rf temp/*
	@echo "清理完成"

# 下载依赖
.PHONY: deps
deps:
	@echo "下载依赖..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "依赖下载完成"

# 格式化代码
.PHONY: fmt
fmt:
	@echo "格式化代码..."
	$(GOFMT) ./...
	@echo "代码格式化完成"

# 代码静态检查
.PHONY: vet
vet:
	@echo "代码静态检查..."
	$(GOVET) ./...
	@echo "静态检查完成"

# 运行测试
.PHONY: test
test:
	@echo "运行测试..."
	$(GOTEST) -v ./...
	@echo "测试完成"

# 运行测试并生成覆盖率报告
.PHONY: test-coverage
test-coverage:
	@echo "运行测试并生成覆盖率报告..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告生成完成: coverage.html"

# 代码质量检查
.PHONY: lint
lint:
	@echo "代码质量检查..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint未安装，跳过代码质量检查"; \
		echo "安装命令: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi
	@echo "代码质量检查完成"

# 运行应用程序
.PHONY: run
run: build
	@echo "运行 $(APP_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

# 开发模式运行
.PHONY: dev
dev:
	@echo "开发模式运行..."
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "air未安装，使用go run运行"; \
		echo "安装air: go install github.com/cosmtrek/air@latest"; \
		$(GOCMD) run cmd/agent/main.go; \
	fi

# Docker构建
.PHONY: docker-build
docker-build:
	@echo "构建Docker镜像..."
	docker build -f docker/Dockerfile -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):$(VERSION)
	@echo "Docker镜像构建完成: $(DOCKER_IMAGE):$(DOCKER_TAG)"

# Docker运行
.PHONY: docker-run
docker-run:
	@echo "运行Docker容器..."
	docker-compose -f docker/docker-compose.yaml up -d
	@echo "Docker容器启动完成"

# 开发环境Docker运行
.PHONY: docker-dev
docker-dev:
	@echo "开发环境Docker运行..."
	docker-compose -f docker/docker-compose.dev.yaml up -d
	@echo "开发环境Docker容器启动完成"

# Docker推送
.PHONY: docker-push
docker-push: docker-build
	@echo "推送Docker镜像..."
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	docker push $(DOCKER_IMAGE):$(VERSION)
	@echo "Docker镜像推送完成"

# 安装到系统
.PHONY: install
install: build
	@echo "安装到系统..."
	@if [ "$(shell uname)" = "Darwin" ] || [ "$(shell uname)" = "Linux" ]; then \
		sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/; \
		sudo chmod +x /usr/local/bin/$(BINARY_NAME); \
		echo "安装完成: /usr/local/bin/$(BINARY_NAME)"; \
	else \
		echo "Windows系统请手动复制 $(BUILD_DIR)/$(BINARY_WINDOWS) 到PATH目录"; \
	fi

# 从系统卸载
.PHONY: uninstall
uninstall:
	@echo "从系统卸载..."
	@if [ "$(shell uname)" = "Darwin" ] || [ "$(shell uname)" = "Linux" ]; then \
		sudo rm -f /usr/local/bin/$(BINARY_NAME); \
		echo "卸载完成"; \
	else \
		echo "Windows系统请手动删除PATH目录中的 $(BINARY_WINDOWS)"; \
	fi

# 显示版本信息
.PHONY: version
version:
	@echo "应用名称: $(APP_NAME)"
	@echo "版本: $(VERSION)"
	@echo "构建时间: $(BUILD_TIME)"
	@echo "Git提交: $(GIT_COMMIT)"
	@echo "Go版本: $(GO_VERSION)"

# 创建发布包
.PHONY: release
release: clean build-all
	@echo "创建发布包..."
	@mkdir -p release
	@cp $(BUILD_DIR)/$(BINARY_UNIX) release/$(APP_NAME)-$(VERSION)-linux-amd64
	@cp $(BUILD_DIR)/$(BINARY_WINDOWS) release/$(APP_NAME)-$(VERSION)-windows-amd64.exe
	@cp $(BUILD_DIR)/$(BINARY_DARWIN) release/$(APP_NAME)-$(VERSION)-darwin-amd64
	@cp configs/config.yaml release/
	@cp .env.example release/
	@echo "发布包创建完成: release/"

# 快速开发流程
.PHONY: quick
quick: fmt vet test build

# 完整CI流程
.PHONY: ci
ci: clean deps fmt vet lint test-coverage build

# 部署相关
.PHONY: deploy
deploy: docker-build docker-push
	@echo "部署完成"

# 监控日志
.PHONY: logs
logs:
	@echo "查看应用日志..."
	@if [ -f "logs/agent.log" ]; then \
		tail -f logs/agent.log; \
	else \
		echo "日志文件不存在: logs/agent.log"; \
	fi

# 停止Docker服务
.PHONY: docker-stop
docker-stop:
	@echo "停止Docker服务..."
	docker-compose -f docker/docker-compose.yaml down
	docker-compose -f docker/docker-compose.dev.yaml down
	@echo "Docker服务已停止"

# 清理Docker资源
.PHONY: docker-clean
docker-clean: docker-stop
	@echo "清理Docker资源..."
	docker system prune -f
	docker volume prune -f
	@echo "Docker资源清理完成"