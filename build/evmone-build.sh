#!/bin/bash
# Build evmone library for go-ethereum integration
# Usage: evmone-build.sh [native|amd64|arm64|mipsle]
set -e

TARGET="${1:-native}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
EVMONE_DIR="$SCRIPT_DIR/../evmone"

# Map Go arch names to build directories and CMake settings
case "$TARGET" in
    native|amd64)
        BUILD_DIR="$EVMONE_DIR/build"
        CMAKE_EXTRA=""
        ;;
    arm64)
        if ! command -v aarch64-linux-gnu-gcc &>/dev/null; then
            echo "Skipping arm64 build: cross-compiler not found"
            exit 0
        fi
        BUILD_DIR="$EVMONE_DIR/build-arm64"
        CMAKE_EXTRA="-DCMAKE_SYSTEM_PROCESSOR=aarch64 -DCMAKE_C_COMPILER=aarch64-linux-gnu-gcc -DCMAKE_CXX_COMPILER=aarch64-linux-gnu-g++"
        ;;
    mipsle)
        if ! command -v mipsel-linux-gnu-gcc &>/dev/null; then
            echo "Skipping mipsle build: cross-compiler not found"
            exit 0
        fi
        BUILD_DIR="$EVMONE_DIR/build-mipsle"
        CMAKE_EXTRA="-DCMAKE_SYSTEM_PROCESSOR=mipsel -DCMAKE_C_COMPILER=mipsel-linux-gnu-gcc -DCMAKE_CXX_COMPILER=mipsel-linux-gnu-g++ -DCMAKE_SYSTEM_NAME=Linux"
        ;;
    *)
        echo "Unknown target: $TARGET" >&2
        exit 1
        ;;
esac

mkdir -p "$BUILD_DIR"
cd "$BUILD_DIR"

cmake "$EVMONE_DIR" \
    -DCMAKE_BUILD_TYPE=Release \
    -DBUILD_SHARED_LIBS=ON \
    -DEVMONE_TESTING=OFF \
    -DEVMONE_FUZZING=OFF \
    -DCMAKE_INSTALL_LIBDIR=lib \
    $CMAKE_EXTRA

cmake --build . --parallel

# Ensure lib/ exists (some distros use lib64/)
if [ -d "$BUILD_DIR/lib64" ] && [ ! -d "$BUILD_DIR/lib" ]; then
    ln -sf lib64 "$BUILD_DIR/lib"
fi
