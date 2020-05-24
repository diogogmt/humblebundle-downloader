BIN_DIR ?= ./bin
BIN_NAME ?= hbd
define build
	go build \
		-tags='$(BUILD_TAGS)' \
		-gcflags='-e' \
		-ldflags='-s -w' \
		-o $(BIN_DIR)/$(BIN_NAME) \
		./cmd/hbd
endef

## build: builds hbd binary
build:
	$(call build)

## imports: runs goimports
imports:
	goimports -w $(shell find . -type f -name '*.go' -not -path "./vendor/*")

## lint: runs golint
lint:
	golint ./...

## test: runs go test
test:
	go test ./...

## vet: runs go vet
vet:
	go vet ./...

## staticcheck: runs staticcheck
staticcheck:
	staticcheck $(shell go list ./...)

.PHONY: vendor
## vendor: updates vendored dependencies
vendor:
	rm -f vendor go.sum || :
	go mod init || :
	go mod tidy
	go mod vendor

## help: prints this help message
help:
	@echo "Usage: \n"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'