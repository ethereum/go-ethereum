#!/bin/sh
#
# Set environment variables so that we can compile rocksdb using
# fbcode settings.  It uses the latest g++ compiler and also
# uses jemalloc

# location of libgcc
LIBGCC_BASE="/mnt/gvfs/third-party2/libgcc/7712e757d7355cb51292454ee0b7b46a467fdfed/4.8.1/gcc-4.8.1-glibc-2.17/8aac7fc"
LIBGCC_INCLUDE="$LIBGCC_BASE/include"
LIBGCC_LIBS=" -L $LIBGCC_BASE/libs"

# location of glibc
GLIBC_REV=6e40560b4e0b6d690fd1cf8c7a43ad7452b04cfa
GLIBC_INCLUDE="/mnt/gvfs/third-party2/glibc/$GLIBC_REV/2.17/gcc-4.8.1-glibc-2.17/99df8fc/include"
GLIBC_LIBS=" -L /mnt/gvfs/third-party2/glibc/$GLIBC_REV/2.17/gcc-4.8.1-glibc-2.17/99df8fc/lib"

# location of snappy headers and libraries
SNAPPY_INCLUDE=" -I /mnt/gvfs/third-party2/snappy/aef17f6c0b44b4fe408bd06f67c93701ab0a6ceb/1.0.3/gcc-4.8.1-glibc-2.17/43d84e2/include"
SNAPPY_LIBS=" /mnt/gvfs/third-party2/snappy/aef17f6c0b44b4fe408bd06f67c93701ab0a6ceb/1.0.3/gcc-4.8.1-glibc-2.17/43d84e2/lib/libsnappy.a"

# location of zlib headers and libraries
ZLIB_INCLUDE=" -I /mnt/gvfs/third-party2/zlib/25c6216928b4d77b59ddeca0990ff6fe9ac16b81/1.2.5/gcc-4.8.1-glibc-2.17/c3f970a/include"
ZLIB_LIBS=" /mnt/gvfs/third-party2/zlib/25c6216928b4d77b59ddeca0990ff6fe9ac16b81/1.2.5/gcc-4.8.1-glibc-2.17/c3f970a/lib/libz.a"

# location of bzip headers and libraries
BZIP_INCLUDE=" -I /mnt/gvfs/third-party2/bzip2/c9ef7629c2aa0024f7a416e87602f06eb88f5eac/1.0.6/gcc-4.8.1-glibc-2.17/c3f970a/include/"
BZIP_LIBS=" /mnt/gvfs/third-party2/bzip2/c9ef7629c2aa0024f7a416e87602f06eb88f5eac/1.0.6/gcc-4.8.1-glibc-2.17/c3f970a/lib/libbz2.a"

LZ4_REV=065ec7e38fe83329031f6668c43bef83eff5808b
LZ4_INCLUDE=" -I /mnt/gvfs/third-party2/lz4/$LZ4_REV/r108/gcc-4.8.1-glibc-2.17/c3f970a/include"
LZ4_LIBS=" /mnt/gvfs/third-party2/lz4/$LZ4_REV/r108/gcc-4.8.1-glibc-2.17/c3f970a/lib/liblz4.a"

ZSTD_REV=8df2d01673ae6afcc8c8d16fec862b2d67ecc1e9
ZSTD_INCLUDE=" -I /mnt/gvfs/third-party2/zstd/$ZSTD_REV/0.1.1/gcc-4.8.1-glibc-2.17/c3f970a/include"
ZSTD_LIBS=" /mnt/gvfs/third-party2/zstd/$ZSTD_REV/0.1.1/gcc-4.8.1-glibc-2.17/c3f970a/lib/libzstd.a"

# location of gflags headers and libraries
GFLAGS_INCLUDE=" -I /mnt/gvfs/third-party2/gflags/1ad047a6e6f6673991918ecadc670868205a243a/1.6/gcc-4.8.1-glibc-2.17/c3f970a/include/"
GFLAGS_LIBS=" /mnt/gvfs/third-party2/gflags/1ad047a6e6f6673991918ecadc670868205a243a/1.6/gcc-4.8.1-glibc-2.17/c3f970a/lib/libgflags.a"

# location of jemalloc
JEMALLOC_INCLUDE=" -I /mnt/gvfs/third-party2/jemalloc/3691c776ac26dd8781e84f8888b6a0fbdbc0a9ed/dev/gcc-4.8.1-glibc-2.17/4d53c6f/include"
JEMALLOC_LIB="/mnt/gvfs/third-party2/jemalloc/3691c776ac26dd8781e84f8888b6a0fbdbc0a9ed/dev/gcc-4.8.1-glibc-2.17/4d53c6f/lib/libjemalloc.a"

# location of numa
NUMA_REV=829d10dac0230f99cd7e1778869d2adf3da24b65
NUMA_INCLUDE=" -I /mnt/gvfs/third-party2/numa/$NUMA_REV/2.0.8/gcc-4.8.1-glibc-2.17/c3f970a/include/"
NUMA_LIB=" /mnt/gvfs/third-party2/numa/$NUMA_REV/2.0.8/gcc-4.8.1-glibc-2.17/c3f970a/lib/libnuma.a"

# location of libunwind
LIBUNWIND_REV=2c060e64064559905d46fd194000d61592087bdc
LIBUNWIND="/mnt/gvfs/third-party2/libunwind/$LIBUNWIND_REV/1.1/gcc-4.8.1-glibc-2.17/675d945/lib/libunwind.a"

# use Intel SSE support for checksum calculations
export USE_SSE=1

BINUTILS="/mnt/gvfs/third-party2/binutils/2aff2e7b474cd3e6ab23495ad1224b7d214b9f8e/2.21.1/centos6-native/da39a3e/bin"
AR="$BINUTILS/ar"

DEPS_INCLUDE="$SNAPPY_INCLUDE $ZLIB_INCLUDE $BZIP_INCLUDE $LZ4_INCLUDE $ZSTD_INCLUDE $GFLAGS_INCLUDE $NUMA_INCLUDE"

GCC_BASE="/mnt/gvfs/third-party2/gcc/1ec615e23800f0815d474478ba476a0adc3fe788/4.8.1/centos6-native/cc6c9dc"
STDLIBS="-L $GCC_BASE/lib64"

if [ -z "$USE_CLANG" ]; then
  # gcc
  CC="$GCC_BASE/bin/gcc"
  CXX="$GCC_BASE/bin/g++"
  
  CFLAGS="-B$BINUTILS/gold -m64 -mtune=generic"
  CFLAGS+=" -isystem $GLIBC_INCLUDE"
  CFLAGS+=" -isystem $LIBGCC_INCLUDE"
else
  # clang 
  CLANG_BASE="/mnt/gvfs/third-party2/clang/9ab68376f938992c4eb5946ca68f90c3185cffc8/3.4"
  CLANG_INCLUDE="$CLANG_BASE/gcc-4.8.1-glibc-2.17/fb0f730/lib/clang/3.4/include"
  CC="$CLANG_BASE/centos6-native/9cefd8a/bin/clang"
  CXX="$CLANG_BASE/centos6-native/9cefd8a/bin/clang++"

  KERNEL_HEADERS_INCLUDE="/mnt/gvfs/third-party2/kernel-headers/a683ed7135276731065a9d76d3016c9731f4e2f9/3.2.18_70_fbk11_00129_gc8882d0/gcc-4.8.1-glibc-2.17/da39a3e/include/"

  CFLAGS="-B$BINUTILS/gold -nostdinc -nostdlib"
  CFLAGS+=" -isystem $LIBGCC_BASE/include/c++/4.8.1 "
  CFLAGS+=" -isystem $LIBGCC_BASE/include/c++/4.8.1/x86_64-facebook-linux "
  CFLAGS+=" -isystem $GLIBC_INCLUDE"
  CFLAGS+=" -isystem $LIBGCC_INCLUDE"
  CFLAGS+=" -isystem $CLANG_INCLUDE"
  CFLAGS+=" -isystem $KERNEL_HEADERS_INCLUDE/linux "
  CFLAGS+=" -isystem $KERNEL_HEADERS_INCLUDE "
  CXXFLAGS="-nostdinc++"
fi

CFLAGS+=" $DEPS_INCLUDE"
CFLAGS+=" -DROCKSDB_PLATFORM_POSIX -DROCKSDB_FALLOCATE_PRESENT -DROCKSDB_MALLOC_USABLE_SIZE"
CFLAGS+=" -DSNAPPY -DGFLAGS=google -DZLIB -DBZIP2 -DLZ4 -DZSTD -DNUMA"
CXXFLAGS+=" $CFLAGS"

EXEC_LDFLAGS=" $SNAPPY_LIBS $ZLIB_LIBS $BZIP_LIBS $LZ4_LIBS $ZSTD_LIBS $GFLAGS_LIBS $NUMA_LIB"
EXEC_LDFLAGS+=" -Wl,--dynamic-linker,/usr/local/fbcode/gcc-4.8.1-glibc-2.17/lib/ld.so"
EXEC_LDFLAGS+=" $LIBUNWIND"
EXEC_LDFLAGS+=" -Wl,-rpath=/usr/local/fbcode/gcc-4.8.1-glibc-2.17/lib"

PLATFORM_LDFLAGS="$LIBGCC_LIBS $GLIBC_LIBS $STDLIBS -lgcc -lstdc++"

EXEC_LDFLAGS_SHARED="$SNAPPY_LIBS $ZLIB_LIBS $BZIP_LIBS $LZ4_LIBS $ZSTD_LIBS $GFLAGS_LIBS"

VALGRIND_REV=b2a9f85e4b70cd03abc85a7f3027fbc4cef35bd0
VALGRIND_VER="/mnt/gvfs/third-party2/valgrind/$VALGRIND_REV/3.8.1/gcc-4.8.1-glibc-2.17/c3f970a/bin/"

export CC CXX AR CFLAGS CXXFLAGS EXEC_LDFLAGS EXEC_LDFLAGS_SHARED VALGRIND_VER JEMALLOC_LIB JEMALLOC_INCLUDE
