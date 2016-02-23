#!/bin/sh

# fail early
set -e

if test -z $ROCKSDB_PATH; then
  ROCKSDB_PATH=~/rocksdb
fi
source $ROCKSDB_PATH/build_tools/fbcode_config4.8.1.sh

EXTRA_LDFLAGS=""

if test -z $ALLOC; then
  # default
  ALLOC=tcmalloc
elif [[ $ALLOC == "jemalloc" ]]; then
  ALLOC=system
  EXTRA_LDFLAGS+=" -Wl,--whole-archive $JEMALLOC_LIB -Wl,--no-whole-archive"
fi

# we need to force mongo to use static library, not shared
STATIC_LIB_DEP_DIR='build/static_library_dependencies'
test -d $STATIC_LIB_DEP_DIR || mkdir $STATIC_LIB_DEP_DIR
test -h $STATIC_LIB_DEP_DIR/`basename $SNAPPY_LIBS` || ln -s $SNAPPY_LIBS $STATIC_LIB_DEP_DIR
test -h $STATIC_LIB_DEP_DIR/`basename $LZ4_LIBS` || ln -s $LZ4_LIBS $STATIC_LIB_DEP_DIR

EXTRA_LDFLAGS+=" -L $STATIC_LIB_DEP_DIR"

set -x

EXTRA_CMD=""
if ! test -e version.json; then
  # this is Mongo 3.0
  EXTRA_CMD="--rocksdb \
    --variant-dir=linux2/norm
    --cxx=${CXX} \
    --cc=${CC} \
    --use-system-zlib"  # add this line back to normal code path
                        # when https://jira.mongodb.org/browse/SERVER-19123 is resolved
fi

scons \
  LINKFLAGS="$EXTRA_LDFLAGS $EXEC_LDFLAGS $PLATFORM_LDFLAGS" \
  CCFLAGS="$CXXFLAGS -L $STATIC_LIB_DEP_DIR" \
  LIBS="lz4 gcc stdc++" \
  LIBPATH="$ROCKSDB_PATH" \
  CPPPATH="$ROCKSDB_PATH/include" \
  -j32 \
  --allocator=$ALLOC \
  --nostrip \
  --opt=on \
  --disable-minimum-compiler-version-enforcement \
  --use-system-snappy \
  --disable-warnings-as-errors \
  $EXTRA_CMD $*
