#!/bin/bash
# XDC Network Test Suite
# This script runs various tests for the XDC Network

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$ROOT_DIR"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}XDC Network Test Suite${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# Parse arguments
VERBOSE=""
RACE=""
COVERAGE=""
BENCHMARK=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--verbose)
            VERBOSE="-v"
            shift
            ;;
        -r|--race)
            RACE="-race"
            shift
            ;;
        -c|--coverage)
            COVERAGE="-coverprofile=coverage.out"
            shift
            ;;
        -b|--benchmark)
            BENCHMARK="true"
            shift
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Function to run tests
run_tests() {
    local package=$1
    local name=$2
    
    echo -e "${YELLOW}Running $name tests...${NC}"
    if go test $VERBOSE $RACE $COVERAGE "$package"; then
        echo -e "${GREEN}✓ $name tests passed${NC}"
    else
        echo -e "${RED}✗ $name tests failed${NC}"
        exit 1
    fi
    echo ""
}

# Core tests
echo -e "${YELLOW}=== Core Tests ===${NC}"
run_tests "./core/..." "Core"

# Consensus tests
echo -e "${YELLOW}=== Consensus Tests ===${NC}"
run_tests "./consensus/XDPoS/..." "XDPoS Consensus"

# eth package tests
echo -e "${YELLOW}=== Eth Package Tests ===${NC}"
run_tests "./eth/..." "Eth"

# p2p tests
echo -e "${YELLOW}=== P2P Tests ===${NC}"
run_tests "./p2p/..." "P2P"

# API tests
echo -e "${YELLOW}=== API Tests ===${NC}"
run_tests "./internal/ethapi/..." "API"

# Contract tests
echo -e "${YELLOW}=== Contract Tests ===${NC}"
run_tests "./contracts/..." "Contracts"

# Run benchmarks if requested
if [ "$BENCHMARK" = "true" ]; then
    echo -e "${YELLOW}=== Running Benchmarks ===${NC}"
    
    echo "XDPoS Consensus Benchmarks:"
    go test -bench=. -benchmem ./consensus/XDPoS/... 2>/dev/null || true
    
    echo ""
    echo "Core Benchmarks:"
    go test -bench=. -benchmem ./core/... 2>/dev/null || true
    
    echo ""
fi

# Coverage report
if [ -n "$COVERAGE" ]; then
    echo -e "${YELLOW}=== Coverage Report ===${NC}"
    go tool cover -func=coverage.out | tail -n 1
    echo ""
    
    # Generate HTML report
    go tool cover -html=coverage.out -o coverage.html
    echo -e "${GREEN}Coverage report generated: coverage.html${NC}"
fi

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}All tests passed!${NC}"
echo -e "${GREEN}========================================${NC}"
