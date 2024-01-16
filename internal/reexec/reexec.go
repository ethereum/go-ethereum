// This file originates from Docker/Moby,
// https://github.com/moby/moby/blob/master/pkg/reexec/reexec.go
// Licensed under Apache License 2.0: https://github.com/moby/moby/blob/master/LICENSE
// Copyright 2013-2018 Docker, Inc.
//
// Package reexec facilitates the busybox style reexec of the docker binary that
// we require because of the forking limitations of using Go.  Handlers can be
// registered with a name and the argv 0 of the exec of the binary will be used
// to find and execute custom init paths.
package reexec

import (
	"fmt"
	"os"
)

var registeredInitializers = make(map[string]func())

// Register adds an initialization func under the specified name
func Register(name string, initializer func()) {
	if _, exists := registeredInitializers[name]; exists {
		panic(fmt.Sprintf("reexec func already registered under name %q", name))
	}
	registeredInitializers[name] = initializer
}

// Init is called as the first part of the exec process and returns true if an
// initialization function was called.
func Init() bool {
	if initializer, ok := registeredInitializers[os.Args[0]]; ok {
		initializer()
		return true
	}
	return false
}
