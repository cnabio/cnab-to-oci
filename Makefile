.DEFAULT_GOAL := all
SHELL:=/bin/bash

PKG_NAME := github.com/cnabio/cnab-to-oci

EXEC_EXT :=
ifeq ($(OS),Windows_NT)
  EXEC_EXT := .exe
endif

ifeq ($(TAG),)
  TAG := $(shell git describe --always --dirty 2> /dev/null)
endif
ifeq ($(COMMIT),)
  COMMIT := $(shell git rev-parse --short HEAD 2> /dev/null)
endif

ifeq ($(BUILDTIME),)
  BUILDTIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ" 2> /dev/null)
endif
ifeq ($(BUILDTIME),)
  BUILDTIME := unknown
  $(warning unable to set BUILDTIME. Set the value manually)
endif

LDFLAGS := "-s -w \
  -X $(PKG_NAME)/internal.GitCommit=$(COMMIT)     \
  -X $(PKG_NAME)/internal.Version=$(TAG)          \
  -X $(PKG_NAME)/internal.BuildTime=$(BUILDTIME)"

BUILD_ARGS := \
  --build-arg BUILDTIME \
  --build-arg COMMIT    \
  --build-arg TAG \
  --build-arg=GOPROXY

GO_BUILD := CGO_ENABLED=0 go build -ldflags=$(LDFLAGS)
GO_TEST := CGO_ENABLED=0 go test -ldflags=$(LDFLAGS) -failfast
GO_TEST_RACE := go test -ldflags=$(LDFLAGS) -failfast -race

all: build test

all-ci: lint all

check_go_env:
	@test $$(go list) = "$(PKG_NAME)" || \
		(echo "Invalid Go environment - The local directory structure must match:  $(PKG_NAME)" && false)

get-tools:
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.63.4

# Default build
build: bin/cnab-to-oci

bin/%: cmd/% check_go_env
	$(GO_BUILD) -o $@$(EXEC_EXT) ./$<

install:
	pushd cmd/cnab-to-oci && go install && popd

clean:
	rm -rf bin

test: test-unit test-e2e

test-unit:
	$(GO_TEST_RACE) $(shell go list ./... | grep -vE '/e2e')

test-e2e: e2e-image
	docker run --rm --network=host -v /var/run/docker.sock:/var/run/docker.sock cnab-to-oci-e2e

build-e2e-test:
	$(GO_TEST) -c github.com/cnabio/cnab-to-oci/e2e

e2e-image:
	docker build $(BUILD_ARGS) . -t cnab-to-oci-e2e

format: get-tools
	go fmt ./...
	@for source in `find . -type f -name '*.go' -not -path "./vendor/*"`; do \
		goimports -w $$source ; \
	done

lint: get-tools
	golangci-lint run ./...

.PHONY: all get-tools build clean test test-unit test-e2e e2e-image lint
