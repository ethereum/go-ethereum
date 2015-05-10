#!/bin/bash

# Strict mode
set -e

VALGRIND_ARGS="--tool=memcheck"
VALGRIND_ARGS+=" --leak-check=yes"
VALGRIND_ARGS+=" --track-origins=yes"
VALGRIND_ARGS+=" --show-reachable=yes"
VALGRIND_ARGS+=" --num-callers=20"
VALGRIND_ARGS+=" --track-fds=yes"

SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ]; do
  DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"
  SOURCE="$(readlink "$SOURCE")"
  [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE"
done
TEST_DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"

rm -rf $TEST_DIR/build 
mkdir -p $TEST_DIR/build 
cd $TEST_DIR/build ; 
cmake ../../.. > /dev/null 
make Test 
./test/c/Test

# If we have valgrind also run memory check tests
if hash valgrind 2>/dev/null; then
	echo "======== Running tests under valgrind ========";
	cd $TEST_DIR/build/ && valgrind $VALGRIND_ARGS ./test/c/Test
fi
