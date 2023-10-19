// This file originates from Docker/Moby,
// https://github.com/moby/moby/blob/master/pkg/reexec/
// Licensed under AGPL V2: https://github.com/moby/moby/blob/master/LICENSE
// Copyright 2013-2018 Docker, Inc.

//go:build linux

package reexec

// Self returns the path to the current process's binary.
// Returns "/proc/self/exe".
func Self() string {
	return "/proc/self/exe"
}
