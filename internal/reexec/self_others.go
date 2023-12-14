// This file originates from Docker/Moby,
// https://github.com/moby/moby/blob/master/pkg/reexec/
// Licensed under Apache License 2.0: https://github.com/moby/moby/blob/master/LICENSE
// Copyright 2013-2018 Docker, Inc.

//go:build !linux

package reexec

import (
	"os"
	"os/exec"
	"path/filepath"
)

// Self returns the path to the current process's binary.
// Uses os.Args[0].
func Self() string {
	name := os.Args[0]
	if filepath.Base(name) == name {
		if lp, err := exec.LookPath(name); err == nil {
			return lp
		}
	}
	// handle conversion of relative paths to absolute
	if absName, err := filepath.Abs(name); err == nil {
		return absName
	}
	// if we couldn't get absolute name, return original
	// (NOTE: Go only errors on Abs() if os.Getwd fails)
	return name
}
