// Package bloomfilter is face-meltingly fast, thread-safe,
// marshalable, unionable, probability- and
// optimal-size-calculating Bloom filter in go
//
// https://github.com/steakknife/bloomfilter
//
// Copyright © 2014, 2015, 2018 Barry Allard
//
// MIT license
//
package bloomfilter

import (
	"log"
	"os"
)

const debugVar = "GOLANG_STEAKKNIFE_BLOOMFILTER_DEBUG"

// EnableDebugging permits debug() logging of details to stderr
func EnableDebugging() {
	err := os.Setenv(debugVar, "1")
	if err != nil {
		panic("Unable to Setenv " + debugVar)
	}
}

func debugging() bool {
	return os.Getenv(debugVar) != ""
}

// debug printing when debugging() is true
func debug(format string, a ...interface{}) {
	if debugging() {
		log.Printf(format, a...)
	}
}
