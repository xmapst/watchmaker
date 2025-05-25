#!/usr/bin/env make -f

EXAMPLES_C = clock_gettime gettimeofday time
SOURCES_C = $(addsuffix .c, $(EXAMPLES_C))

build: $(EXAMPLES_C)

$(EXAMPLES_C): $(SOURCES_C)
	cc -o $@ $@.c

.PHONY: clean
clean:
	rm -f $(EXAMPLES_C)
