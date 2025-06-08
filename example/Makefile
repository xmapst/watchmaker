#!/usr/bin/env make -f

EXAMPLES_CPP = print_time
SOURCES_CPP = $(addsuffix .cpp, $(EXAMPLES_CPP))

EXAMPLES_C = clock_gettime gettimeofday time
SOURCES_C = $(addsuffix .c, $(EXAMPLES_C))

EXAMPLES_GO = gotime
SOURCES_GO = $(addsuffix .go, $(EXAMPLES_GO))

build: $(EXAMPLES_CPP) $(EXAMPLES_C) $(EXAMPLES_GO)

$(EXAMPLES_CPP): $(SOURCES_CPP)
	g++ -o $@ $@.cpp -lpthread

$(EXAMPLES_C): $(SOURCES_C)
	gcc -o $@ $@.c

$(EXAMPLES_GO): $(SOURCES_GO)
	go build $@.go

.PHONY: clean
clean:
	rm -f $(EXAMPLES_CPP) $(EXAMPLES_C) $(EXAMPLES_GO)
