#!/bin/bash
#A shell script for Jenknis to run valgrind on rocksdb tests
#Returns 0 on success when there are no failed tests 

VALGRIND_DIR=build_tools/VALGRIND_LOGS
make clean
make -j$(nproc) valgrind_check
NUM_FAILED_TESTS=$((`wc -l $VALGRIND_DIR/valgrind_failed_tests | awk '{print $1}'` - 1))
if [ $NUM_FAILED_TESTS -lt 1 ]; then
  echo No tests have valgrind errors
  exit 0
else
  cat $VALGRIND_DIR/valgrind_failed_tests
  exit 1
fi
