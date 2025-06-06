#!/usr/bin/env make -f

TOPDIR := $(realpath $(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
SELF := $(abspath $(lastword $(MAKEFILE_LIST)))

GIT_URL := $(shell git remote -v|grep push|awk '{print $$2}')
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
GIT_COMMIT := $(shell git rev-parse HEAD)
BUILD_TIME := $(shell date +"%Y-%m-%d %H:%M:%S %Z")
LDFLAGS := "-w -s"

ARCH := $(shell uname -m)

CFLAGS := -fPIE -O2

OBJ_SRCS_amd64 := fake_clock_gettime fake_gettimeofday fake_time
# on modern arm64 kernels time() works via gettimeofday()
OBJ_SRCS_arm64 := fake_clock_gettime fake_gettimeofday

COMPRESS_GO_BINARIES ?= 0

ifeq ($(COMPRESS_GO_BINARIES),1)
ifndef UPX
ifeq ($(shell upx --version >/dev/null 2>&1 || echo FAIL),)
UPX = upx
endif
endif
UPX ?= $(error upx not found)
endif

.PHONY: help
help: ## Show help message (list targets)
	@awk 'BEGIN {FS = ":.*##"; printf "\nTargets:\n"} /^[$$()% 0-9a-zA-Z_-]+:.*?##/ {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(SELF)

show-var-%:
	@{ \
	escaped_v="$(subst ",\",$($*))" ; \
	if [ -n "$$escaped_v" ]; then v="$$escaped_v"; else v="(undefined)"; fi; \
	printf "%-19s %s\n" "$*" "$$v"; \
	}

SHOW_ENV_VARS = \
	TOPDIR \
	SELF \
	GIT_URL \
	GIT_BRANCH \
	GIT_COMMIT \
	BUILD_TIME \
	LDFLAGS \
	ARCH \
	CFLAGS \
	OBJ_SRCS_amd64 \
	OBJ_SRCS_arm64 \
	COMPRESS_GO_BINARIES

show-env: $(addprefix show-var-, $(SHOW_ENV_VARS)) ## Show environment details

.PHONY: build all_build init_env build_amd64 build_arm64
build: ## Build amd64 and arm64 binaries via Docker
	@echo "===> Building watchmaker on $(ARCH) host..."
	@docker run -it --rm --network host -v $(shell pwd):/go/src/watchmaker -w /go/src/watchmaker golang:latest make all_build

all_build: init_env build_amd64 build_arm64 ## Install depdendncies and build amd64/arm64 binaries on Linux host

.PHONY: build-env
build-env: ## Run building environment in Docker container
	@echo "===> Running build env on $(ARCH) host..."
	docker run -it --rm --network host -v $(shell pwd):/go/src/watchmaker -w /go/src/watchmaker golang:latest /bin/bash

examples: ## Build examples
	$(MAKE) -C example

.PHONY: init_env_amd64
init_env_amd64: ## Install dependencies to amd64/x86_64 host
	@echo "===> Initializing environment ($(ARCH))..."
	@apt update && apt install -y gcc-12-aarch64-linux-gnu git tar xz-utils \
	&& wget https://github.com/upx/upx/releases/download/v5.0.0/upx-5.0.0-amd64_linux.tar.xz \
	&& tar -xf upx-5.0.0-amd64_linux.tar.xz \
	&& mv upx-5.0.0-amd64_linux/upx /usr/local/bin/upx \
	&& rm -rf upx-5.0.0-amd64_linux*

.PHONY: init_env_arm64
init_env_arm64: ## Install dependencies to arm64/aarch64 host
	@echo "===> Initializing environment ($(ARCH))..."
	@apt update && apt install -y gcc-12-x86-64-linux-gnu git tar xz-utils \
	&& wget https://github.com/upx/upx/releases/download/v5.0.0/upx-5.0.0-arm64_linux.tar.xz \
	&& tar -xf upx-5.0.0-arm64_linux.tar.xz \
	&& mv upx-5.0.0-arm64_linux/upx /usr/local/bin/upx \
	&& rm -rf upx-5.0.0-arm64_linux*

init_env_x86_64: init_env_amd64
init_env_aarch64: init_env_arm64

init_env: init_env_$(ARCH) ## Install dependencies (auto-detect host arch)

.PHONY: build_amd64_amd64
build_amd64_amd64: ## Build amd64 binaries on amd64/x86_64 host
	@{ \
	echo "===> Building watchmaker_linux_amd64 on $(ARCH)..." ; \
	set -ex ; \
	for src in $(OBJ_SRCS_amd64); do \
		gcc -c fakeclock/$${src}.c $(CFLAGS) -o fakeclock/$${src}_amd64.o ; \
		objdump -x fakeclock/$${src}_amd64.o ; \
	done ; \
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags $(LDFLAGS) -o bin/watchmaker_linux_amd64 ./cmd/... ; \
	}
ifeq ($(COMPRESS_GO_BINARIES),1)
	$(UPX) --force-overwrite --lzma bin/watchmaker_linux_amd64
endif

.PHONY: build_amd64_arm64
build_amd64_arm64: ## Build amd64 binaries on arm64/aarch64 host
	@{ \
	echo "===> Building watchmaker_linux_amd64 on $(ARCH)..." ; \
	set -ex ; \
	for src in $(OBJ_SRCS_amd64); do \
		x86_64-linux-gnu-gcc-12 -c fakeclock/$${src}.c $(CFLAGS) -o fakeclock/$${src}_amd64.o ; \
		x86_64-linux-gnu-objdump -x fakeclock/$${src}_amd64.o ; \
	done ; \
	CGO_ENABLED=0 CC=x86_64-linux-gnu-gcc-12 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags $(LDFLAGS) -o bin/watchmaker_linux_amd64 ./cmd/... ; \
	}
ifeq ($(COMPRESS_GO_BINARIES),1)
	$(UPX) --lzma bin/watchmaker_linux_amd64
endif

build_amd64_x86_64: build_amd64_amd64
build_amd64_aarch64: build_amd64_arm64

build_amd64: build_amd64_$(ARCH) ## Build amd64 binaries (auto-detect host arch)

.PHONY: build_arm64_amd64
build_arm64_amd64: ## Build arm64 binaries on amd64/x86_64 host
	@{ \
	echo "===> Building watchmaker_linux_arm64 on $(ARCH)..." ; \
	set -ex ; \
	for src in $(OBJ_SRCS_arm64); do \
		aarch64-linux-gnu-gcc-12 -c fakeclock/$${src}.c $(CFLAGS) -o fakeclock/$${src}_arm64.o ; \
		aarch64-linux-gnu-objdump -x fakeclock/$${src}_arm64.o ; \
	done; \
	CGO_ENABLED=0 CC=aarch64-linux-gnu-gcc-12 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags $(LDFLAGS) -o bin/watchmaker_linux_arm64 ./cmd/... ; \
	}
ifeq ($(COMPRESS_GO_BINARIES),1)
	$(UPX) --lzma bin/watchmaker_linux_arm64
endif

.PHONY: build_arm64_arm64
build_arm64_arm64: ## Build arm64 binaries on arm64/aarch64 host
	@{ \
	echo "===> Building watchmaker_linux_arm64 on $(ARCH)..." ; \
	set -ex ; \
	for src in $(OBJ_SRCS_arm64); do \
		gcc -c fakeclock/$${src}.c $(CFLAGS) -o fakeclock/$${src}_arm64.o ; \
		objdump -x fakeclock/$${src}_arm64.o ; \
	done ; \
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags $(LDFLAGS) -o bin/watchmaker_linux_arm64 ./cmd/... ; \
	}
ifeq ($(COMPRESS_GO_BINARIES),1)
	$(UPX) --lzma bin/watchmaker_linux_arm64
endif

build_arm64_x86_64: build_arm64_amd64
build_arm64_aarch64: build_arm64_arm64

build_arm64: build_arm64_$(ARCH) ## Build arm64 binaries (auto-detect host arch)

.PHONY: clean
clean: ## Clean up
	rm -f fakeclock/*.o
	rm -rf bin
	$(MAKE) -C example clean
