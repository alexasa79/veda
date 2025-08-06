# Go Test Project

This directory contains a Go project used for testing LSP integration features in Veda.

## Purpose

The files in this directory are used by integration tests to verify that LSP features work correctly, particularly:
- Code completion after unsaved changes (didChange notifications)
- Signature help
- Go to definition
- Document symbols

## Files

- `main.go` - A simple Go program with functions used for testing LSP features

## Usage

This project is used by tests in `internal/proxy/lsp_didchange_test.go` and other LSP integration tests.

**Important**: These files should not be modified manually as they are used by automated tests. The test suite will temporarily modify these files during testing but will restore them to their original state after completion.