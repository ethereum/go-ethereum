package clipboard

// #cgo CPPFLAGS: -I./
// #cgo CXXFLAGS: -std=c++0x -pedantic-errors -Wall -fno-strict-aliasing
// #cgo LDFLAGS: -lstdc++
// #cgo pkg-config: Qt5Quick
//
// #include "capi.hpp"
import "C"

import "github.com/obscuren/qml"

func SetQMLClipboard(context *qml.Context) {
	context.SetVar("clipboard", (unsafe.Pointer)(C.initClipboard()))
}
