PKG_NAME := github.com/docker/cnab-to-oci
PKG_PATH := /go/src/$(PKG_NAME)
BIN_NAME ?= cnab-to-oci
E2E_NAME := $(BIN_NAME)-e2e

EXEC_EXT :=
ifeq ($(OS),Windows_NT)
  EXEC_EXT := .exe
endif

# Failing to resolve sh.exe to a full path denotes a windows vanilla shell.
# Although 'simple' commands are still exec'ed, 'complex' ones are batch'ed instead of sh'ed.
ifeq ($(SHELL),sh.exe)
  mkdir = mkdir $(subst /,\,$(1)) > nul 2>&1 || (exit 0)
  rm = del /F /Q $(subst /,\,$(1)) > nul 2>&1 || (exit 0)
  rmdir = rmdir /S /Q $(subst /,\,$(1)) > nul 2>&1 || (exit 0)
  chmod =
  BUILDTIME ?= unknown
  NULL := nul
else
  # The no-op redirection forces make to shell out the commands instead of spawning a process as
  # the latter can fail on windows running cmd or powershell while having a unix style shell in the path.
  mkdir = mkdir -p $(1) 1>&1
  rm = rm -rf $(1) 1>&1
  rmdir = rm -rf $(1) 1>&1
  chmod = chmod $(1) $(2) 1>&1
  NULL := /dev/null
endif

ifeq ($(BUILD_TAG),)
  BUILD_TAG := $(shell git describe --always --dirty --abbrev=10 2> $(NULL))
endif
ifeq ($(TAG),)
  ifeq ($(TAG_NAME),)
    TAG := $(BUILD_TAG)
  else
    TAG := $(TAG_NAME)
  endif
endif
ifeq ($(COMMIT),)
  COMMIT := $(shell git rev-parse --short HEAD 2> $(NULL))
endif

ifeq ($(BUILDTIME),)
  BUILDTIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ" 2> $(NULL))
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
  --build-arg BUILDTIME=$(BUILDTIME) \
  --build-arg COMMIT=$(COMMIT)       \
  --build-arg TAG=$(TAG)

GO_BUILD := CGO_ENABLED=0 go build -ldflags=$(LDFLAGS)
GO_TEST := CGO_ENABLED=0 go test -ldflags=$(LDFLAGS) -failfast
GO_TEST_RACE := go test -ldflags=$(LDFLAGS) -failfast -race
