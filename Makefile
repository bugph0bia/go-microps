# Makefile
.DEFAULT_GOAL := build

BINARY_NAME=test
BUILD_DIR=test
TARGET_PATH=./test/
BUILD_TAGS=$(if $(TAGS),-tags $(TAGS),)

FLAGS = -ldflags="-s -w" -trimpath

fmt:
	go fmt $(TARGET_PATH)
.PHONY: fmt

lint: fmt
	staticcheck $(TARGET_PATH)
.PHONY: lint

vet: fmt
	go vet $(TARGET_PATH)
.PHONY: vet

build: vet
	go mod tidy
	go build $(FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(BUILD_TAGS) $(TARGET_PATH)
.PHONY: build
