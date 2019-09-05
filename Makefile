BRANCH?=$(shell git symbolic-ref --short HEAD)
REVISION?=$(shell git rev-parse --short HEAD)

GOFLAGS=-tags=netgo
GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)
PKG=hellorouter

OUT?=bin/$(GOOS)-$(GOARCH)/$(PKG)

build:
	$(info building for $(GOOS)-$(GOARCH) to $(OUT))
	@go build -gcflags '-m=0' -ldflags '-X main.revision=$(REVISION) -X main.branch=$(BRANCH)' -o $(OUT) ./internal/cmd/main.go

build_router: export GOOS=linux
build_router: export GOARCH=mipsle
build_router: build

build_arm: export GOOS=linux
build_arm: export GOARCH=arm
build_arm: export GOARM=6
build_arm: build

check:
	$(info checking for errors)
	golangci-lint run

test:
	$(info testing)
	go test -timeout=60s -count=1 -race ./internal/...
