#!/bin/bash

# Development helper script for c8y-session-1password
# This script provides common development tasks

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."
    
    # Check Go
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed"
        exit 1
    fi
    print_success "Go $(go version | cut -d' ' -f3) is installed"
    
    # Check 1Password CLI
    if ! command -v op &> /dev/null; then
        print_warning "1Password CLI (op) is not installed"
        print_status "Install from: https://developer.1password.com/docs/cli/"
    else
        print_success "1Password CLI is installed"
    fi
    
    # Check if signed in to 1Password
    if command -v op &> /dev/null; then
        if ! op account list &> /dev/null; then
            print_warning "Not signed in to 1Password CLI"
            print_status "Run: op signin"
        else
            print_success "Signed in to 1Password CLI"
        fi
    fi
}

# Setup development environment
setup() {
    print_status "Setting up development environment..."
    
    # Download dependencies
    go mod download
    go mod tidy
    
    print_success "Development environment ready"
}

# Run tests
test() {
    print_status "Running tests..."
    go test -v -race ./...
    print_success "All tests passed"
}

# Run tests with coverage
coverage() {
    print_status "Running tests with coverage..."
    go test -v -race -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    print_success "Coverage report generated: coverage.html"
}

# Build the project
build() {
    print_status "Building project..."
    make build
    print_success "Build complete: c8y-session-1password"
}

# Run linting
lint() {
    print_status "Running linting..."
    if command -v golangci-lint &> /dev/null; then
        golangci-lint run
        print_success "Linting complete"
    else
        print_error "golangci-lint is not installed"
        print_status "Install from: https://golangci-lint.run/usage/install/"
        exit 1
    fi
}

# Clean build artifacts
clean() {
    print_status "Cleaning build artifacts..."
    make clean
    print_success "Clean complete"
}

# Interactive test with 1Password
test_interactive() {
    print_status "Testing interactive mode..."
    
    if ! command -v op &> /dev/null; then
        print_error "1Password CLI is required for interactive testing"
        exit 1
    fi
    
    if ! op account list &> /dev/null; then
        print_error "Please sign in to 1Password CLI first: op signin"
        exit 1
    fi
    
    build
    print_status "Running interactive picker..."
    ./c8y-session-1password list
}

# Show help
show_help() {
    echo "Development helper script for c8y-session-1password"
    echo ""
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  setup      Setup development environment"
    echo "  test       Run tests"
    echo "  coverage   Run tests with coverage"
    echo "  build      Build the project"
    echo "  lint       Run linting"
    echo "  clean      Clean build artifacts"
    echo "  check      Check prerequisites"
    echo "  interactive Test interactive mode with 1Password"
    echo "  all        Run test, lint, and build"
    echo "  help       Show this help"
    echo ""
    echo "Examples:"
    echo "  $0 setup"
    echo "  $0 test"
    echo "  $0 all"
}

# Run all checks
run_all() {
    check_prerequisites
    test
    lint
    build
    print_success "All checks passed!"
}

# Main script logic
case "$1" in
    setup)
        setup
        ;;
    test)
        test
        ;;
    coverage)
        coverage
        ;;
    build)
        build
        ;;
    lint)
        lint
        ;;
    clean)
        clean
        ;;
    check)
        check_prerequisites
        ;;
    interactive)
        test_interactive
        ;;
    all)
        run_all
        ;;
    help|--help|-h)
        show_help
        ;;
    "")
        show_help
        ;;
    *)
        print_error "Unknown command: $1"
        show_help
        exit 1
        ;;
esac
