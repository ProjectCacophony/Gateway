.PHONY: all clean build lint

NAME ?= gateway
SOURCE ?= ./cmd/gateway

GOOS ?= linux
GOARCH ?= amd64
BUILD_DIR ?= bin/$(GOOS).$(GOARCH)

BINARY= $(BUILD_DIR)/$(NAME)
BUILD_FLAGS=

SOURCE_FOLDERS := $(shell go list -f {{.Dir}} ./...)

all: build

clean:
	rm -Rf bin/

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v $(BUILD_FLAGS) -o "$(BINARY)" $(SOURCE)

build_windows:
	export GOOS=windows
	go build -v -o "bin/windows.$(GOARCH)/$(NAME).exe" $(SOURCE)

lint:
	golangci-lint run --deadline=30m --disable-all \
	--enable=govet \
	--enable=staticcheck \
	--enable=unused \
	--enable=gosimple \
	--enable=structcheck \
	--enable=varcheck \
	--enable=ineffassign \
	--enable=deadcode \
	--enable=golint \
	--enable=unconvert \
	--enable=goimports \
	--enable=maligned \
	--enable=unparam \
	--enable=prealloc \
	--enable=scopelint \
	./...
