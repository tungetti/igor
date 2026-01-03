#!/bin/bash
#
# test.sh - Test script for Igor
#
# This script runs linting, tests, and generates coverage reports.
#

set -e

# Change to project root
cd "$(dirname "$0")/.."

echo "========================================="
echo "Running Igor Test Suite"
echo "========================================="
echo ""

# Run go vet
echo "Running go vet..."
go vet ./...
echo "✓ go vet passed"
echo ""

# Run go fmt check
echo "Checking code formatting..."
UNFORMATTED=$(gofmt -l . 2>&1 || true)
if [ -n "$UNFORMATTED" ]; then
    echo "Warning: The following files are not properly formatted:"
    echo "$UNFORMATTED"
    echo "Run 'gofmt -w .' to fix formatting."
else
    echo "✓ Code formatting check passed"
fi
echo ""

# Run tests with coverage
echo "Running tests with coverage..."
go test -v -race -coverprofile=coverage.out ./...
echo ""

# Display coverage summary if coverage file exists
if [ -f coverage.out ]; then
    echo "Coverage Summary:"
    go tool cover -func=coverage.out | tail -n 1
    
    # Generate HTML coverage report
    go tool cover -html=coverage.out -o coverage.html
    echo "HTML coverage report generated: coverage.html"
fi

echo ""
echo "========================================="
echo "All tests completed successfully!"
echo "========================================="
