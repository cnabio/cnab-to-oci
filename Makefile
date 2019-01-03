.DEFAULT_GOAL := all
SHELL:=/bin/bash

PKG_NAME := github.com/docker/cnab-to-oci

EXEC_EXT :=
ifeq ($(OS),Windows_NT)
  EXEC_EXT := .exe
endif

GO_BUILD := CGO_ENABLED=0 go build

all: build test

all-ci: lint all

check_go_env:
	@test $$(go list) = "$(PKG_NAME)" || \
		(echo "Invalid Go environment - The local directory structure must match:  $(PKG_NAME)" && false)

get-tools:
	go get golang.org/x/tools/cmd/goimports
	go get github.com/alecthomas/gometalinter
	gometalinter --install

# Default build
build:
	make bin/cnab-to-oci

bin/%: cmd/% check_go_env
	$(GO_BUILD) -o $@$(EXEC_EXT) ./$<

install:
	pushd cmd/cnab-to-oci && go install && popd

clean:
	rm -rf bin

test:
	go test -failfast ./...

format: get-tools
	go fmt ./...
	@for source in `find . -type f -name '*.go' -not -path "./vendor/*"`; do \
		goimports -w $$source ; \
	done

lint: get-tools
	gometalinter --config=gometalinter.json ./...

.PHONY: all, get-tools, build, clean, test, lint
