.PHONY: all build clean install test lint fmt help

# 变量
APP_NAME := fcapital
VERSION := 1.0.0
BUILD_DIR := build
CMD_DIR := cmd/fcapital
GO := go
GOFLAGS := -v

# 构建信息
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(GIT_COMMIT) -X main.date=$(BUILD_TIME)"

# 默认目标
all: build

# 帮助
help:
	@echo "fcapital - A Comprehensive Penetration Testing Framework"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build       Build the application"
	@echo "  install     Install dependencies"
	@echo "  run         Run the application"
	@echo "  test        Run tests"
	@echo "  lint        Run linter"
	@echo "  fmt         Format code"
	@echo "  clean       Clean build artifacts"
	@echo "  cross       Cross-compile for multiple platforms"
	@echo "  help        Show this help message"

# 安装依赖
install:
	@echo "[*] Installing dependencies..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "[+] Dependencies installed"

# 构建
build:
	@echo "[*] Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) ./$(CMD_DIR)
	@echo "[+] Build complete: $(BUILD_DIR)/$(APP_NAME)"

# 运行
run: build
	@echo "[*] Running $(APP_NAME)..."
	./$(BUILD_DIR)/$(APP_NAME)

# 测试
test:
	@echo "[*] Running tests..."
	$(GO) test -v -race -coverprofile=coverage.out ./...
	@echo "[+] Tests complete"

# 代码检查
lint:
	@echo "[*] Running linter..."
	@which golangci-lint > /dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run ./...
	@echo "[+] Lint complete"

# 格式化
fmt:
	@echo "[*] Formatting code..."
	$(GO) fmt ./...
	@echo "[+] Format complete"

# 清理
clean:
	@echo "[*] Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out
	@echo "[+] Clean complete"

# 交叉编译
cross:
	@echo "[*] Cross-compiling..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 ./$(CMD_DIR)
	GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64 ./$(CMD_DIR)
	GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 ./$(CMD_DIR)
	GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 ./$(CMD_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe ./$(CMD_DIR)
	@echo "[+] Cross-compilation complete"

# 开发模式 (热重载)
dev:
	@echo "[*] Starting development mode..."
	@which air > /dev/null || go install github.com/cosmtop/air@latest
	air

# 生成文档
docs:
	@echo "[*] Generating documentation..."
	$(GO) doc -all ./... > docs/api.md
	@echo "[+] Documentation generated"
