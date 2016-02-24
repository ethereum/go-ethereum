#!/bin/sh
if [ "$#" = "0" ]; then
  echo "Usage: $0 major|minor|patch"
  exit 1
fi
if [ "$1" = "major" ]; then
  cat include/rocksdb/version.h  | grep MAJOR | head -n1 | awk '{print $3}'
fi
if [ "$1" = "minor" ]; then
  cat include/rocksdb/version.h  | grep MINOR | head -n1 | awk '{print $3}'
fi
if [ "$1" = "patch" ]; then
  cat include/rocksdb/version.h  | grep PATCH | head -n1 | awk '{print $3}'
fi
