#!/bin/sh
#
# Set environment variables so that we can compile rocksdb using
# fbcode settings.  It uses the latest g++ and clang compilers and also
# uses jemalloc
# Environment variables that change the behavior of this script:
# PIC_BUILD -- if true, it will only take pic versions of libraries from fbcode. libraries that don't have pic variant will not be included

CFLAGS=""

# location of libgcc
LIBGCC_BASE="/mnt/gvfs/third-party2/libgcc/0473c80518a10d6efcbe24c5eeca3fb4ec9b519c/4.9.x/gcc-4.9-glibc-2.20/e1a7e4e"
LIBGCC_INCLUDE="$LIBGCC_BASE/include"
LIBGCC_LIBS=" -L $LIBGCC_BASE/libs"

# location of glibc
GLIBC_REV=7397bed99280af5d9543439cdb7d018af7542720
GLIBC_INCLUDE="/mnt/gvfs/third-party2/glibc/$GLIBC_REV/2.20/gcc-4.9-glibc-2.20/99df8fc/include"
GLIBC_LIBS=" -L /mnt/gvfs/third-party2/glibc/$GLIBC_REV/2.20/gcc-4.9-glibc-2.20/99df8fc/lib"

SNAPPY_INCLUDE=" -I /mnt/gvfs/third-party2/snappy/b0f269b3ca47770121aa159b99e1d8d2ab260e1f/1.0.3/gcc-4.9-glibc-2.20/c32916f/include/"

if test -z $PIC_BUILD; then
  SNAPPY_LIBS=" /mnt/gvfs/third-party2/snappy/b0f269b3ca47770121aa159b99e1d8d2ab260e1f/1.0.3/gcc-4.9-glibc-2.20/c32916f/lib/libsnappy.a"
else
  SNAPPY_LIBS=" /mnt/gvfs/third-party2/snappy/b0f269b3ca47770121aa159b99e1d8d2ab260e1f/1.0.3/gcc-4.9-glibc-2.20/c32916f/lib/libsnappy_pic.a"
fi

CFLAGS+=" -DSNAPPY"

if test -z $PIC_BUILD; then
  # location of zlib headers and libraries
  ZLIB_INCLUDE=" -I /mnt/gvfs/third-party2/zlib/feb983d9667f4cf5e9da07ce75abc824764b67a1/1.2.8/gcc-4.9-glibc-2.20/4230243/include/"
  ZLIB_LIBS=" /mnt/gvfs/third-party2/zlib/feb983d9667f4cf5e9da07ce75abc824764b67a1/1.2.8/gcc-4.9-glibc-2.20/4230243/lib/libz.a"
  CFLAGS+=" -DZLIB"

  # location of bzip headers and libraries
  BZIP_INCLUDE=" -I /mnt/gvfs/third-party2/bzip2/af004cceebb2dfd173ca29933ea5915e727aad2f/1.0.6/gcc-4.9-glibc-2.20/4230243/include/"
  BZIP_LIBS=" /mnt/gvfs/third-party2/bzip2/af004cceebb2dfd173ca29933ea5915e727aad2f/1.0.6/gcc-4.9-glibc-2.20/4230243/lib/libbz2.a"
  CFLAGS+=" -DBZIP2"

  LZ4_INCLUDE=" -I /mnt/gvfs/third-party2/lz4/79d2943e2dd7208a3e0b06cf95e9f85f05fe9e1b/r124/gcc-4.9-glibc-2.20/4230243/include/"
  LZ4_LIBS=" /mnt/gvfs/third-party2/lz4/79d2943e2dd7208a3e0b06cf95e9f85f05fe9e1b/r124/gcc-4.9-glibc-2.20/4230243/lib/liblz4.a"
  CFLAGS+=" -DLZ4"

  ZSTD_REV=8df2d01673ae6afcc8c8d16fec862b2d67ecc1e9
  ZSTD_INCLUDE=" -I /mnt/gvfs/third-party2/zstd/$ZSTD_REV/0.1.1/gcc-4.8.1-glibc-2.17/c3f970a/include"
  ZSTD_LIBS=" /mnt/gvfs/third-party2/zstd/$ZSTD_REV/0.1.1/gcc-4.8.1-glibc-2.17/c3f970a/lib/libzstd.a"
  CFLAGS+=" -DZSTD"
fi

# location of gflags headers and libraries
GFLAGS_INCLUDE=" -I /mnt/gvfs/third-party2/gflags/0fa60e2b88de3e469db6c482d6e6dac72f5d65f9/1.6/gcc-4.9-glibc-2.20/4230243/include/"
if test -z $PIC_BUILD; then
  GFLAGS_LIBS=" /mnt/gvfs/third-party2/gflags/0fa60e2b88de3e469db6c482d6e6dac72f5d65f9/1.6/gcc-4.9-glibc-2.20/4230243/lib/libgflags.a"
else
  GFLAGS_LIBS=" /mnt/gvfs/third-party2/gflags/0fa60e2b88de3e469db6c482d6e6dac72f5d65f9/1.6/gcc-4.9-glibc-2.20/4230243/lib/libgflags_pic.a"
fi
CFLAGS+=" -DGFLAGS=google"

# location of jemalloc
JEMALLOC_INCLUDE=" -I /mnt/gvfs/third-party2/jemalloc/bcd68e5e419efa4e61b9486d6854564d6d75a0b5/3.6.0/gcc-4.9-glibc-2.20/2aafc78/include/"
JEMALLOC_LIB=" /mnt/gvfs/third-party2/jemalloc/bcd68e5e419efa4e61b9486d6854564d6d75a0b5/3.6.0/gcc-4.9-glibc-2.20/2aafc78/lib/libjemalloc.a"

if test -z $PIC_BUILD; then
  # location of numa
  NUMA_INCLUDE=" -I /mnt/gvfs/third-party2/numa/bbefc39ecbf31d0ca184168eb613ef8d397790ee/2.0.8/gcc-4.9-glibc-2.20/4230243/include/"
  NUMA_LIB=" /mnt/gvfs/third-party2/numa/bbefc39ecbf31d0ca184168eb613ef8d397790ee/2.0.8/gcc-4.9-glibc-2.20/4230243/lib/libnuma.a"
  CFLAGS+=" -DNUMA"

  # location of libunwind
  LIBUNWIND="/mnt/gvfs/third-party2/libunwind/1de3b75e0afedfe5585b231bbb340ec7a1542335/1.1/gcc-4.9-glibc-2.20/34235e8/lib/libunwind.a"
fi

# use Intel SSE support for checksum calculations
export USE_SSE=1

BINUTILS="/mnt/gvfs/third-party2/binutils/0b6ad0c88ddd903333a48ae8bff134efac468e4a/2.25/centos6-native/da39a3e/bin"
AR="$BINUTILS/ar"

DEPS_INCLUDE="$SNAPPY_INCLUDE $ZLIB_INCLUDE $BZIP_INCLUDE $LZ4_INCLUDE $ZSTD_INCLUDE $GFLAGS_INCLUDE $NUMA_INCLUDE"

GCC_BASE="/mnt/gvfs/third-party2/gcc/1c67a0b88f64d4d9ced0382d141c76aaa7d62fba/4.9.x/centos6-native/1317bc4"
STDLIBS="-L $GCC_BASE/lib64"

CLANG_BASE="/mnt/gvfs/third-party2/clang/d81444dd214df3d2466734de45bb264a0486acc3/dev"
CLANG_BIN="$CLANG_BASE/centos6-native/af4b1a0/bin"
CLANG_ANALYZER="$CLANG_BIN/clang++"
CLANG_SCAN_BUILD="$CLANG_BASE/src/clang/tools/scan-build/scan-build"

if [ -z "$USE_CLANG" ]; then
  # gcc
  CC="$GCC_BASE/bin/gcc"
  CXX="$GCC_BASE/bin/g++"
  
  CFLAGS+=" -B$BINUTILS/gold"
  CFLAGS+=" -isystem $GLIBC_INCLUDE"
  CFLAGS+=" -isystem $LIBGCC_INCLUDE"
else
  # clang 
  CLANG_INCLUDE="$CLANG_BASE/gcc-4.9-glibc-2.20/74c386f/lib/clang/dev/include/"
  CC="$CLANG_BIN/clang"
  CXX="$CLANG_BIN/clang++"

  KERNEL_HEADERS_INCLUDE="/mnt/gvfs/third-party2/kernel-headers/ffd14f660a43c4b92717986b1bba66722ef089d0/3.2.18_70_fbk11_00129_gc8882d0/gcc-4.9-glibc-2.20/da39a3e/include"

  CFLAGS+=" -B$BINUTILS/gold -nostdinc -nostdlib"
  CFLAGS+=" -isystem $LIBGCC_BASE/include/c++/4.9.x "
  CFLAGS+=" -isystem $LIBGCC_BASE/include/c++/4.9.x/x86_64-facebook-linux "
  CFLAGS+=" -isystem $GLIBC_INCLUDE"
  CFLAGS+=" -isystem $LIBGCC_INCLUDE"
  CFLAGS+=" -isystem $CLANG_INCLUDE"
  CFLAGS+=" -isystem $KERNEL_HEADERS_INCLUDE/linux "
  CFLAGS+=" -isystem $KERNEL_HEADERS_INCLUDE "
  CXXFLAGS="-nostdinc++"
fi

CFLAGS+=" $DEPS_INCLUDE"
CFLAGS+=" -DROCKSDB_PLATFORM_POSIX -DROCKSDB_FALLOCATE_PRESENT -DROCKSDB_MALLOC_USABLE_SIZE"
CXXFLAGS+=" $CFLAGS"

EXEC_LDFLAGS=" $SNAPPY_LIBS $ZLIB_LIBS $BZIP_LIBS $LZ4_LIBS $ZSTD_LIBS $GFLAGS_LIBS $NUMA_LIB"
EXEC_LDFLAGS+=" -Wl,--dynamic-linker,/usr/local/fbcode/gcc-4.9-glibc-2.20/lib/ld.so"
EXEC_LDFLAGS+=" $LIBUNWIND"
EXEC_LDFLAGS+=" -Wl,-rpath=/usr/local/fbcode/gcc-4.9-glibc-2.20/lib"

PLATFORM_LDFLAGS="$LIBGCC_LIBS $GLIBC_LIBS $STDLIBS -lgcc -lstdc++"

EXEC_LDFLAGS_SHARED="$SNAPPY_LIBS $ZLIB_LIBS $BZIP_LIBS $LZ4_LIBS $ZSTD_LIBS $GFLAGS_LIBS"

VALGRIND_VER="/mnt/gvfs/third-party2/valgrind/6c45ef049cbf11c2df593addb712cd891049e737/3.10.0/gcc-4.9-glibc-2.20/4230243/bin/"

export CC CXX AR CFLAGS CXXFLAGS EXEC_LDFLAGS EXEC_LDFLAGS_SHARED VALGRIND_VER JEMALLOC_LIB JEMALLOC_INCLUDE CLANG_ANALYZER CLANG_SCAN_BUILD
