#!/usr/bin/env make -f

EXAMPLES_C = clock_gettime gettimeofday time
SOURCES_C = $(addsuffix .c, $(EXAMPLES_C))

EXAMPLES_GO = gotime
SOURCES_GO = $(addsuffix .go, $(EXAMPLES_GO))

build: $(EXAMPLES_C) $(EXAMPLES_GO)

$(EXAMPLES_C): $(SOURCES_C)
	cc -o $@ $@.c

$(EXAMPLES_GO): $(SOURCES_GO)
	go build $@.go

.PHONY: clean
clean:
	rm -f $(EXAMPLES_C) $(EXAMPLES_GO)
