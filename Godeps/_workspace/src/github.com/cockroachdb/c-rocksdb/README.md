# RocksDB

This is a go-gettable version of the RocksDB C++ library for use in Go code that needs to link
against the C++ rocksdb library but wants to integrate with `go get` and `go build`. The protobuf
source is currently pinned to the 2.6.1 release.

To use in your project you need to import the package and set appropriate cgo flag directives:

```
import _ "github.com/cockroachdb/c-rocksdb"

// #cgo CXXFLAGS: -std=c++11
// #cgo CPPFLAGS: -I <relative-path>/c-rocksdb/internal/include
// #cgo darwin LDFLAGS: -Wl,-undefined -Wl,dynamic_lookup
// #cgo !darwin LDFLAGS: -Wl,-unresolved-symbols=ignore-all
import "C"
```

To update the upstream version of RocksDB you'll want to update `./import.sh`
to point to the new version (just change the URL), and then run it. Make sure
the CockroachDB tests still pass!
