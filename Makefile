include vars.mk

.DEFAULT_GOAL := all
SHELL:=/bin/bash

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
build: bin/$(BIN_NAME)

cross: bin/$(BIN_NAME)-linux bin/$(BIN_NAME)-darwin bin/$(BIN_NAME)-windows.exe

bin/$(BIN_NAME): cmd/$(BIN_NAME) check_go_env
	$(GO_BUILD) -o $@$(EXEC_EXT) ./$<

bin/$(BIN_NAME)-%.exe bin/$(BIN_NAME)-%: cmd/$(BIN_NAME) check_go_env
	GOOS=$* $(GO_BUILD) -o $@ ./$<
	
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
	$(GO_TEST) -c github.com/docker/cnab-to-oci/e2e

e2e-image:
	docker build $(BUILD_ARGS) . -t cnab-to-oci-e2e

format: get-tools
	go fmt ./...
	@for source in `find . -type f -name '*.go' -not -path "./vendor/*"`; do \
		goimports -w $$source ; \
	done

lint: get-tools
	gometalinter --config=gometalinter.json ./...

.PHONY: all, get-tools, build, clean, test, test-unit, test-e2e, e2e-image, lint
