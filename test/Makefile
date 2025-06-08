#!/usr/bin/env make -f

TOPDIR := $(realpath $(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
SELF := $(abspath $(lastword $(MAKEFILE_LIST)))

TESTS_C = test_clock_gettime test_gettimeofday test_time
SOURCES_C = $(addsuffix .c, $(TESTS_C))

build: $(TESTS_C)

$(TESTS_C): $(SOURCES_C)
	gcc -o $@ $@.c

run-test-%: $(TESTS_C)
	$(TOPDIR)/runtest.sh "$*" "$(TOPDIR)"

test: run-test-clock_gettime run-test-gettimeofday run-test-time

.PHONY: clean
clean:
	rm -f $(TESTS_C)
