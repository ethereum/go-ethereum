# Snappy

This is a fork of the snappy-go library from code.google.com. It has been
changed to use the C++ snappy library for encoding and decoding.

This package is also a go-gettable version of the C++ snappy library for use in
Go code that needs to link against the C++ snappy library but wants to
integrate with `go get` and `go build`. The snappy source is currently pinned
to the 1.1.1 release.

To use in your project you need to import the package and set appropriate cgo
flag directives:

```
import _ "github.com/cockroachdb/c-snappy"

// #cgo CXXFLAGS: -std=c++11
// #cgo CPPFLAGS: -I <relative-path>/c-snappy/internal
// #cgo darwin LDFLAGS: -Wl,-undefined -Wl,dynamic_lookup
// #cgo !darwin LDFLAGS: -Wl,-unresolved-symbols=ignore-all
import "C"
```
