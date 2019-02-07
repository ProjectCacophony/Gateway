.PHONY: all clean build

NAME ?= gateway
SOURCE ?= ./cmd/gateway

GOOS ?= linux
GOARCH ?= amd64
BUILD_DIR ?= bin/$(GOOS).$(GOARCH)

BINARY= $(BUILD_DIR)/$(NAME)
BUILD_FLAGS=

all: build

clean:
	rm -Rf bin/

build:
	go build -v $(BUILD_FLAGS) -o "$(BINARY)" $(SOURCE)
