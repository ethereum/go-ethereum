// Package lz4 uses the cgo compilation facilities to build the
// LZ4 library.
package lz4

// #cgo CPPFLAGS: -Iinternal/lib
import "C"
