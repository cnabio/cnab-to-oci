include vars.mk

LINT_IMAGE_NAME := $(BIN_NAME)-lint:$(TAG)
DEV_IMAGE_NAME := $(BIN_NAME)-dev:$(TAG)
E2E_IMAGE_NAME := $(BIN_NAME)-e2e:$(TAG)

BIN_CTNR_NAME := $(BIN_NAME)-bin-$(TAG)

.DEFAULT: all
all: build test

create_bin:
	@$(call mkdir,bin)

build_dev_image:
	docker build $(BUILD_ARGS) --target=build -t $(DEV_IMAGE_NAME) .

build_e2e_image:
	docker build $(BUILD_ARGS) --target=e2e -t $(E2E_IMAGE_NAME) .

build: create_bin build_dev_image
	docker create --name $(BIN_CTNR_NAME) $(DEV_IMAGE_NAME) noop
	docker cp $(BIN_CTNR_NAME):$(PKG_PATH)/bin/$(BIN_NAME)-linux bin/$(BIN_NAME)-linux
	docker cp $(BIN_CTNR_NAME):$(PKG_PATH)/bin/$(BIN_NAME)-darwin bin/$(BIN_NAME)-darwin
	docker cp $(BIN_CTNR_NAME):$(PKG_PATH)/bin/$(BIN_NAME)-windows.exe bin/$(BIN_NAME)-windows.exe
	docker rm $(BIN_CTNR_NAME)
	@$(call chmod,+x,bin/$(BIN_NAME)-linux)
	@$(call chmod,+x,bin/$(BIN_NAME)-darwin)
	@$(call chmod,+x,bin/$(BIN_NAME)-windows.exe)

shell: build_dev_image ## run a shell in the docker build image
	docker run -ti --rm $(DEV_IMAGE_NAME) bash

test: test-unit test-e2e ## run all tests

test-unit: build_dev_image ## run unit tests
	docker run --rm $(DEV_IMAGE_NAME) make test-unit

test-e2e: build_e2e_image ## run e2e tests
	docker run --rm -v /var/run:/var/run:ro --network="host" $(E2E_IMAGE_NAME)

lint: ## run linter(s)
	$(info Linting...)
	docker build -t $(LINT_IMAGE_NAME) -f lint.Dockerfile .
	docker run --rm $(LINT_IMAGE_NAME) gometalinter --config=gometalinter.json ./...

clean-images: ## Delete images
	docker image rm -f $(DEV_IMAGE_NAME)
	docker image rm -f $(E2E_IMAGE_NAME)

.PHONY: lint test-e2e test-unit test shell build gradle-test shell build_e2e_image build_dev_image create_bin
