SHELL=/bin/bash
GIT_URL := $(shell git remote -v|grep push|awk '{print $$2}')
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
GIT_COMMIT := $(shell git rev-parse HEAD)
BUILD_TIME := $(shell date +"%Y-%m-%d %H:%M:%S %Z")
LDFLAGS := "-w -s \
-X 'github.com/xmapst/watchmaker/internal/version.GitUrl=$(GIT_URL)' \
-X 'github.com/xmapst/watchmaker/internal/version.GitBranch=$(GIT_BRANCH)' \
-X 'github.com/xmapst/watchmaker/internal/version.GitCommit=$(GIT_COMMIT)' \
-X 'github.com/xmapst/watchmaker/internal/version.BuildDate=$(BUILD_TIME)' \
"

build:
	CGO_ENABLED=1 go build -trimpath -ldflags $(LDFLAGS) -o bin/watchmaker ./cmd/...

all: c build

c:
	cc -c ./internal/time/fakeclock/fake_clock_gettime.c -fPIE -O2 -o internal/time/fakeclock/fake_clock_gettime.o
	cc -c ./internal/time/fakeclock/fake_gettimeofday.c -fPIE -O2 -o internal/time/fakeclock/fake_gettimeofday.o