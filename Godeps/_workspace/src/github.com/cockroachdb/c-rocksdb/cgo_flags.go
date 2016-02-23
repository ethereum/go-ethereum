// Package rocksdb uses the cgo compilation facilities to build the
// RocksDB C++ library. Note that support for bzip2 and zlib is not
// compiled in.
package rocksdb

import (
	// explicit because these Go libraries do not export any Go symbols.
	_ "github.com/cockroachdb/c-lz4"
	_ "github.com/cockroachdb/c-snappy"
)

// #cgo CPPFLAGS: -Iinternal -Iinternal/include -Iinternal/db -Iinternal/util
// #cgo CPPFLAGS: -Iinternal/utilities/merge_operators/string_append
// #cgo CPPFLAGS: -I../c-snappy/internal -I../c-lz4/internal/lib
// #cgo CPPFLAGS: -DROCKSDB_PLATFORM_POSIX -DNDEBUG -DSNAPPY -DLZ4
// #cgo darwin CPPFLAGS: -DOS_MACOSX
// #cgo linux CPPFLAGS: -DOS_LINUX
// #cgo freebsd CPPFLAGS: -DOS_FREEBSD
// #cgo dragonfly CPPFLAGS: -DOS_DRAGONFLY
// #cgo CXXFLAGS: -W -Wextra -Wall -Wsign-compare -Wshadow -Wno-unused-parameter
// #cgo CXXFLAGS: -std=c++11 -fno-omit-frame-pointer -momit-leaf-frame-pointer
// #cgo darwin CXXFLAGS: -Wshorten-64-to-32
// #cgo freebsd CXXFLAGS: -Wshorten-64-to-32
// #cgo dragonfly CXXFLAGS: -Wshorten-64-to-32
// #cgo darwin LDFLAGS: -Wl,-undefined -Wl,dynamic_lookup
// #cgo !darwin LDFLAGS: -Wl,-unresolved-symbols=ignore-all
import "C"
