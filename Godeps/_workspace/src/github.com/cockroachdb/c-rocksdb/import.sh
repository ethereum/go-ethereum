#!/usr/bin/env sh

set -eu

rm -rf *.cc internal/*
curl -sL https://github.com/facebook/rocksdb/archive/v4.0.tar.gz | tar zxf - -C internal --strip-components=1
make -C internal util/build_version.cc
patch -p1 < gitignore.patch

# symlink so cgo compiles them
for source_file in $(make sources | grep -vE '(/redis/|_(cmd|tool).cc$)'); do
  ln -sf $source_file .
done

# restore the repo to what it would look like when first cloned.
# comment this line out while updating upstream.
git clean -dxf
