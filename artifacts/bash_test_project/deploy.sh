#!/bin/bash

log_message() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

check_dependencies() {
    local missing=0
    for cmd in git docker curl; do
        if ! command -v "$cmd" &> /dev/null; then
            log_message "Missing dependency: $cmd"
            missing=1
        fi
    done
    return $missing
}

build_project() {
    log_message "Building project..."
    make clean
    make all
}

run_tests() {
    log_message "Running tests..."
    make test
}

deploy() {
    log_message "Starting deployment"
    check_dependencies
    build_project
    run_tests
    log_message "Deployment complete"
}

deploy
