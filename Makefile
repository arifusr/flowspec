BINARY_NAME=apitest
BUILD_DIR=bin
SRC_DIR=src
VERSION=0.1.0

.PHONY: build clean test lint run-example

build:
	@cd $(SRC_DIR) && go build -ldflags "-s -w" -o ../$(BUILD_DIR)/$(BINARY_NAME) ./cmd/main.go
	@echo "✓ Built $(BUILD_DIR)/$(BINARY_NAME)"

build-dev:
	@cd $(SRC_DIR) && go build -o ../$(BUILD_DIR)/$(BINARY_NAME) ./cmd/main.go

clean:
	@rm -rf $(BUILD_DIR)/
	@echo "✓ Cleaned"

test:
	@cd $(SRC_DIR) && go test ./...

vet:
	@cd $(SRC_DIR) && go vet ./...

run-example:
	@cd examples && ../$(BUILD_DIR)/$(BINARY_NAME) run flows/smoke.flow --env dev -v

lint-example:
	@cd examples && ../$(BUILD_DIR)/$(BINARY_NAME) dsl lint .

install: build
	@cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME) 2>/dev/null || \
		cp $(BUILD_DIR)/$(BINARY_NAME) ~/bin/$(BINARY_NAME) 2>/dev/null || \
		echo "Copy $(BUILD_DIR)/$(BINARY_NAME) to your PATH manually"
	@echo "✓ Installed"
