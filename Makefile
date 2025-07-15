APP_NAME := git-branches
BIN_DIR := $(HOME)/go/bin
BUILD_DIR := ./bin

.PHONY: all build install clean

all: build install

build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(APP_NAME)

i: install
install:
	go mod tidy

deploy: build
	@echo "Installing $(APP_NAME) to $(BIN_DIR)..."
	@mkdir -p $(BIN_DIR)
	@cp $(BUILD_DIR)/$(APP_NAME) $(BIN_DIR)/

clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)