# Running unit tests

Follow these steps to run unit tests, in the repo's root dir:

```
docker pull scrolltech/go-rust-builder:go-1.21-rust-nightly-2023-12-03 --platform linux/amd64
docker run -it --rm -v "$(PWD):/workspace" -w /workspace scrolltech/go-rust-builder:go-1.21-rust-nightly-2023-12-03

export LIBSCROLL_ZSTD_VERSION=v0.1.0-rc0-ubuntu20.04
export SCROLL_LIB_PATH=/scroll/lib

mkdir -p $SCROLL_LIB_PATH
wget -O $SCROLL_LIB_PATH/libscroll_zstd.so https://github.com/scroll-tech/da-codec/releases/download/$LIBSCROLL_ZSTD_VERSION/libscroll_zstd.so
export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:$SCROLL_LIB_PATH
export CGO_LDFLAGS="-L$SCROLL_LIB_PATH -Wl,-rpath,$SCROLL_LIB_PATH"
go test -v -race ./rollup/rollup_sync_service/...
```
