# Makefile for veda editor

# Binary name
BINARY=vd

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean

.PHONY: all build test clean

# Default target
all: build

# Build the editor
build:
	$(GOBUILD) -o $(BINARY) main.go

rebuild:
	$(GOBUILD) -a -o $(BINARY) main.go

# Run all tests
# -p sets the number of test packages to run in parallel (default is GOMAXPROCS)
# -parallel sets the number of tests to run in parallel within each package
# You can override parallelism with: make test PARALLEL=8
PARALLEL ?= 16
test:
	$(GOTEST) -v -p $(PARALLEL) -parallel $(PARALLEL) ./internal/*/

testall:
	$(GOTEST) -v -p $(PARALLEL) -parallel $(PARALLEL) -count=1 ./internal/*/

testfast:
	$(GOTEST) -v -p $(PARALLEL) -parallel $(PARALLEL) -failfast ./internal/*/

# Clean up
clean:
	$(GOCLEAN)
	rm -f $(BINARY)
