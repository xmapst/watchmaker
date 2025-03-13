GIT_URL := $(shell git remote -v|grep push|awk '{print $$2}')
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
GIT_COMMIT := $(shell git rev-parse HEAD)
BUILD_TIME := $(shell date +"%Y-%m-%d %H:%M:%S %Z")
LDFLAGS := "-w -s"

.PHONY: build all_build init_env build_amd64 build_arm64
build:
	@echo "Building watchmaker..."
	@docker run -it --rm --network host -v $(shell pwd):/go/src/watchmaker -w /go/src/watchmaker golang:latest make all_build

all_build: init_env build_amd64 build_arm64

init_env:
	@echo "Initializing environment..."
	@apt update && apt install -y gcc-aarch64-linux-gnu git tar xz-utils \
	&& wget https://github.com/upx/upx/releases/download/v4.2.4/upx-4.2.4-amd64_linux.tar.xz \
	&& tar -xf upx-4.2.4-amd64_linux.tar.xz \
	&& mv upx-4.2.4-amd64_linux/upx /usr/local/bin/upx \
	&& rm -rf upx-4.2.4-amd64_linux*

build_amd64:
	@echo "Building watchmaker_linux_amd64..."
	@cc -c fakeclock/fake_clock_gettime.c -fPIE -O2 -o fakeclock/fake_clock_gettime_amd64.o
	@cc -c fakeclock/fake_gettimeofday.c -fPIE -O2 -o fakeclock/fake_gettimeofday_amd64.o
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags $(LDFLAGS) -o bin/watchmaker_linux_amd64 ./cmd/...
	@upx --lzma bin/watchmaker_linux_amd64

build_arm64:
	@echo "Building watchmaker_linux_arm64..."
	@aarch64-linux-gnu-gcc -c fakeclock/fake_clock_gettime.c -fPIE -O2 -o fakeclock/fake_clock_gettime_arm64.o
	@aarch64-linux-gnu-gcc -c fakeclock/fake_gettimeofday.c -fPIE -O2 -o fakeclock/fake_gettimeofday_arm64.o
	@CGO_ENABLED=0 CC=aarch64-linux-gnu-gcc GOOS=linux GOARCH=arm64 go build -trimpath -ldflags $(LDFLAGS) -o bin/watchmaker_linux_arm64 ./cmd/...
	@upx --lzma bin/watchmaker_linux_arm64