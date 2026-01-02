# Makefile
.DEFAULT_GOAL := build

BINARY_NAME=test
BUILD_DIR=test
TARGET_PATH=./test/
BUILD_TAGS=$(if $(TAGS),-tags "$(TAGS)",)
FLAGS = -ldflags="-s -w" -trimpath

TAPDEV = tap0
TAPADDR = 192.0.2.1/24

.PHONY: fmt lint vet build tap

fmt:
	go fmt $(TARGET_PATH)

lint: fmt
	staticcheck $(TARGET_PATH)

vet: fmt
	go vet $(TARGET_PATH)

build: vet
	go mod tidy
	go build $(FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(BUILD_TAGS) $(TARGET_PATH)

tap:
	@ip addr show $(TAPDEV) 2>/dev/null || (echo "Create '$(TAPDEV)'"; \
	  sudo ip tuntap add mode tap user $(USER) name $(TAPDEV); \
	  sudo sysctl -w net.ipv6.conf.$(TAPDEV).disable_ipv6=1; \
	  sudo ip addr add $(TAPADDR) dev $(TAPDEV); \
	  sudo ip link set $(TAPDEV) up; \
	  ip addr show $(TAPDEV); \
	)
