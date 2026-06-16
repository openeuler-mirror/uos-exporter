# 项目信息
PROJECT_NAME := uos-exporter
NAME := dhcpd-leases-exporter
VERSION := 1.0.0
BUILD_DIR := build
BIN_DIR := $(BUILD_DIR)/bin
SRC_DIR := .
MODULES_DIR := modules

# 编译选项
GO := go
GO_BUILD := $(GO) build
GO_CLEAN := $(GO) clean
GO_MOD := $(GO) mod
GO_VET := $(GO) vet
GO_FMT := $(GO) fmt
GO_GET := $(GO) get
GO_FLAGS := -ldflags="-s -w -X main.Version=$(VERSION)"

# 安装路径（支持自定义，默认为 /usr/local/bin）
PREFIX ?= /usr/local
INSTALL_BIN_DIR := $(PREFIX)/bin
INSTALL_UNIT_DIR := /usr/lib/systemd/system
INSTALL_CONFIG_DIR := /etc/uos-exporter

# 构建目标
.PHONY: all build clean deps fmt vet install docker help run build-all

# 默认目标：构建项目
all: deps fmt vet build

# 创建构建目录
$(BIN_DIR):
	mkdir -p $(BIN_DIR)

before_build:
	@echo "Installing dependencies..."
	$(GO_MOD) tidy

# 构建主程序
build: $(BIN_DIR) before_build
	@echo "Building $(NAME)..."
	$(GO_BUILD) $(GO_FLAGS) -o $(BIN_DIR)/$(NAME) $(SRC_DIR)

# 安装到指定路径
install: build
	@echo "Installing $(NAME) to $(DESTDIR)$(INSTALL_BIN_DIR)..."
	install -D -p -m 0755 $(BIN_DIR)/$(NAME) $(DESTDIR)$(INSTALL_BIN_DIR)/$(NAME)
	install -D -p -m 0644 $(SRC_DIR)/$(NAME).service $(DESTDIR)$(INSTALL_UNIT_DIR)/uos-$(NAME).service
	install -D -p -m 0644 $(SRC_DIR)/config/$(NAME).yaml $(DESTDIR)$(INSTALL_CONFIG_DIR)/$(NAME).yaml

# 清理生成的文件
clean:
	@echo "Cleaning up..."
	$(GO_CLEAN)
	rm -rf $(BUILD_DIR)

# 代码格式化
fmt:
	@echo "Formatting code..."
	$(GO_FMT) ./...

# 代码静态检查
vet:
	@echo "Vetting code..."
	$(GO_VET) ./...

# 获取依赖
deps:
	@echo "Getting dependencies..."
	$(GO_MOD) tidy
	$(GO_GET) -v -t ./...

# 构建 Docker 镜像
docker:
	@echo "Building Docker image..."
	docker build -t $(PROJECT_NAME):$(VERSION) .

# 运行程序（开发模式）
run:
	@echo "Running $(NAME) in development mode..."
	$(GO) run . --test-mode

# 构建所有平台的二进制文件
build-all: $(BIN_DIR)
	@echo "Building for multiple platforms..."
	GOOS=linux GOARCH=amd64 $(GO_BUILD) $(GO_FLAGS) -o $(BIN_DIR)/$(NAME)-linux-amd64 $(SRC_DIR)
	GOOS=linux GOARCH=arm64 $(GO_BUILD) $(GO_FLAGS) -o $(BIN_DIR)/$(NAME)-linux-arm64 $(SRC_DIR)
	GOOS=darwin GOARCH=amd64 $(GO_BUILD) $(GO_FLAGS) -o $(BIN_DIR)/$(NAME)-darwin-amd64 $(SRC_DIR)
	GOOS=darwin GOARCH=arm64 $(GO_BUILD) $(GO_FLAGS) -o $(BIN_DIR)/$(NAME)-darwin-arm64 $(SRC_DIR)
	GOOS=windows GOARCH=amd64 $(GO_BUILD) $(GO_FLAGS) -o $(BIN_DIR)/$(NAME)-windows-amd64.exe $(SRC_DIR)

# 提供帮助信息
help:
	@echo "使用方法: make [目标]"
	@echo "可用目标:"
	@echo "  all       - 构建项目（默认）"
	@echo "  build     - 编译主程序"
	@echo "  install   - 安装二进制文件到 /usr/local/bin"
	@echo "  clean     - 删除生成的文件"
	@echo "  fmt       - 格式化代码"
	@echo "  vet       - 静态代码检查"
	@echo "  deps      - 获取依赖"
	@echo "  docker    - 构建 Docker 镜像"
	@echo "  run       - 运行程序（开发模式）"
	@echo "  build-all - 构建所有平台的二进制文件"
	@echo "  help      - 显示帮助信息"
