#!/usr/bin/env sh

set -eu

rm -rf *.c internal/*
curl -sL https://github.com/Cyan4973/lz4/archive/r131.tar.gz | tar zxf - -C internal --strip-components=1

# symlink so cgo compiles them
# files taken from internal/lib/Makefile
for source_file in lz4.c lz4hc.c lz4frame.c xxhash.c; do
  ln -sf internal/lib/$source_file .
done

# restore the repo to what it would look like when first cloned.
# comment this line out while updating upstream.
git clean -dxf
