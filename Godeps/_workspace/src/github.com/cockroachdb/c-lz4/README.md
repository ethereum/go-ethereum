# LZ4

This is a go-gettable version of the LZ4 library for use in Go code that needs to link
against the LZ4 library but wants to integrate with `go get` and `go build`. The LZ4
source is currently pinned to the r128 release.

To use in your project you need to import the package and set appropriate cgo flag directives:

```
import _ "github.com/cockroachdb/c-lz4"

// #cgo CPPFLAGS: -I <relative-path>/c-lz4/internal/lib
// #cgo darwin LDFLAGS: -Wl,-undefined -Wl,dynamic_lookup
// #cgo !darwin LDFLAGS: -Wl,-unresolved-symbols=ignore-all
import "C"
```
