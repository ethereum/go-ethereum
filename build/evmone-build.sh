#!/bin/bash
# Build evmone library for go-ethereum integration
# Usage: evmone-build.sh [native|amd64|arm64|mipsle]
set -e

TARGET="${1:-native}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
EVMONE_DIR="$SCRIPT_DIR/../evmone"

# Detect mipsel cross-compiler: prefer mipsel-linux-gnu-gcc,
# fall back to mips64-linux-gnu-gcc with -EL -mabi=32 flags.
find_mipsel_cc() {
    if command -v mipsel-linux-gnu-gcc &>/dev/null; then
        echo "mipsel-linux-gnu-gcc"
    elif command -v mips64-linux-gnu-gcc &>/dev/null; then
        echo "mips64-linux-gnu-gcc"
    else
        return 1
    fi
}

find_mipsel_cxx() {
    if command -v mipsel-linux-gnu-g++ &>/dev/null; then
        echo "mipsel-linux-gnu-g++"
    elif command -v mips64-linux-gnu-g++ &>/dev/null; then
        echo "mips64-linux-gnu-g++"
    else
        return 1
    fi
}

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
        MIPSEL_CC=$(find_mipsel_cc) || { echo "Skipping mipsle build: cross-compiler not found"; exit 0; }
        MIPSEL_CXX=$(find_mipsel_cxx) || { echo "Skipping mipsle build: cross-compiler not found"; exit 0; }
        BUILD_DIR="$EVMONE_DIR/build-mipsle"
        MIPS_FLAGS=""
        # mips64 compiler needs explicit flags for little-endian 32-bit
        if [[ "$MIPSEL_CC" == *mips64* ]]; then
            MIPS_FLAGS="-EL -mabi=32"
        fi
        CMAKE_EXTRA="-DCMAKE_SYSTEM_NAME=Linux -DCMAKE_SYSTEM_PROCESSOR=mipsel"
        CMAKE_EXTRA="$CMAKE_EXTRA -DCMAKE_C_COMPILER=$MIPSEL_CC -DCMAKE_CXX_COMPILER=$MIPSEL_CXX"
        if [ -n "$MIPS_FLAGS" ]; then
            CMAKE_EXTRA="$CMAKE_EXTRA -DCMAKE_C_FLAGS=$MIPS_FLAGS -DCMAKE_CXX_FLAGS=$MIPS_FLAGS"
        fi
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
