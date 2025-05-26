GIT_URL := $(shell git remote -v|grep push|awk '{print $$2}')
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
GIT_COMMIT := $(shell git rev-parse HEAD)
BUILD_TIME := $(shell date +"%Y-%m-%d %H:%M:%S %Z")
LDFLAGS := "-w -s"

ARCH := $(shell uname -m)

.PHONY: build all_build init_env build_amd64 build_arm64
build:
	@echo "===> Building watchmaker on $(ARCH) host..."
	@docker run -it --rm --network host -v $(shell pwd):/go/src/watchmaker -w /go/src/watchmaker golang:latest make all_build

all_build: init_env build_amd64 build_arm64

.PHONY: build-env
build-env:
	@echo "===> Running build env on $(ARCH) host..."
	docker run -it --rm --network host -v $(shell pwd):/go/src/watchmaker -w /go/src/watchmaker golang:latest /bin/bash

examples:
	$(MAKE) -C example

.PHONY: init_env_x86_64
init_env_x86_64:
	@echo "===> Initializing environment ($(ARCH))..."
	@apt update && apt install -y gcc-12-aarch64-linux-gnu git tar xz-utils \
	&& wget https://github.com/upx/upx/releases/download/v5.0.0/upx-5.0.0-amd64_linux.tar.xz \
	&& tar -xf upx-5.0.0-amd64_linux.tar.xz \
	&& mv upx-5.0.0-amd64_linux/upx /usr/local/bin/upx \
	&& rm -rf upx-5.0.0-amd64_linux*

.PHONY: init_env_arm64
init_env_arm64:
	@echo "===> Initializing environment ($(ARCH))..."
	@apt update && apt install -y gcc-12-x86-64-linux-gnu git tar xz-utils \
	&& wget https://github.com/upx/upx/releases/download/v5.0.0/upx-5.0.0-arm64_linux.tar.xz \
	&& tar -xf upx-5.0.0-arm64_linux.tar.xz \
	&& mv upx-5.0.0-arm64_linux/upx /usr/local/bin/upx \
	&& rm -rf upx-5.0.0-arm64_linux*

init_env_aarch64: init_env_arm64

init_env: init_env_$(ARCH)

.PHONY: build_amd64_x86_64
build_amd64_x86_64:
	@echo "===> Building watchmaker_linux_amd64 on $(ARCH)..."
	gcc -c fakeclock/fake_clock_gettime.c -fPIE -O2 -o fakeclock/fake_clock_gettime_amd64.o
	gcc -c fakeclock/fake_gettimeofday.c -fPIE -O2 -o fakeclock/fake_gettimeofday_amd64.o
	gcc -c fakeclock/fake_time.c -fPIE -O2 -o fakeclock/fake_time_amd64.o
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags $(LDFLAGS) -o bin/watchmaker_linux_amd64 ./cmd/...
	upx --lzma bin/watchmaker_linux_amd64

.PHONY: build_amd64_arm64
build_amd64_arm64:
	@echo "===> Building watchmaker_linux_amd64 on $(ARCH)..."
	x86_64-linux-gnu-gcc-12 -c fakeclock/fake_clock_gettime.c -fPIE -O2 -o fakeclock/fake_clock_gettime_amd64.o
	x86_64-linux-gnu-gcc-12 -c fakeclock/fake_gettimeofday.c -fPIE -O2 -o fakeclock/fake_gettimeofday_amd64.o
	x86_64-linux-gnu-gcc-12 -c fakeclock/fake_time.c -fPIE -O2 -o fakeclock/fake_time_amd64.o
	CGO_ENABLED=0 CC=x86_64-linux-gnu-gcc-12 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags $(LDFLAGS) -o bin/watchmaker_linux_amd64 ./cmd/...
	upx --lzma bin/watchmaker_linux_amd64

build_amd64_aarch64: build_amd64_arm64

build_amd64: build_amd64_$(ARCH)

.PHONY: build_arm64_x86_64
build_arm64_x86_64:
	@echo "===> Building watchmaker_linux_arm64 on $(ARCH)..."
	aarch64-linux-gnu-gcc-12 -c fakeclock/fake_clock_gettime.c -fPIE -O2 -o fakeclock/fake_clock_gettime_arm64.o
	aarch64-linux-gnu-gcc-12 -c fakeclock/fake_gettimeofday.c -fPIE -O2 -o fakeclock/fake_gettimeofday_arm64.o
	aarch64-linux-gnu-gcc-12 -c fakeclock/fake_time.c -fPIE -O2 -o fakeclock/fake_time_arm64.o
	CGO_ENABLED=0 CC=aarch64-linux-gnu-gcc-12 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags $(LDFLAGS) -o bin/watchmaker_linux_arm64 ./cmd/...
	upx --lzma bin/watchmaker_linux_arm64

.PHONY: build_arm64_arm64
build_arm64_arm64:
	@echo "===> Building watchmaker_linux_arm64 on $(ARCH)..."
	gcc -c fakeclock/fake_clock_gettime.c -fPIE -O2 -o fakeclock/fake_clock_gettime_arm64.o
	gcc -c fakeclock/fake_gettimeofday.c -fPIE -O2 -o fakeclock/fake_gettimeofday_arm64.o
	gcc -c fakeclock/fake_time.c -fPIE -O2 -o fakeclock/fake_time_arm64.o
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags $(LDFLAGS) -o bin/watchmaker_linux_arm64 ./cmd/...
	upx --lzma bin/watchmaker_linux_arm64

build_arm64_aarch64: build_arm64_arm64

build_arm64: build_arm64_$(ARCH)

clean:
	rm -f fakeclock/*.o
	rm -rf bin
	$(MAKE) -C example clean
