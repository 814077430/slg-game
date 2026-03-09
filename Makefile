.PHONY: all proto build run clean

all: build

# 生成protobuf代码
proto:
	@echo "Generating protobuf code..."
	@protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/*.proto

# 编译项目
build: proto
	@echo "Building game server..."
	@go build -o slg-server .

# 运行服务器
run: build
	@echo "Starting game server..."
	@./slg-server

# 清理
clean:
	@echo "Cleaning..."
	@rm -f slg-server
	@rm -rf proto/proto/

# 安装依赖
deps:
	@echo "Installing dependencies..."
	@go mod tidy