# Makefile for veda editor

# Binary name
BINARY=veda

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

# Run all tests
test:
	$(GOTEST) -v ./...

# Clean up
clean:
	$(GOCLEAN)
	rm -f $(BINARY)