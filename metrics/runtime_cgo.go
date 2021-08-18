//go:build cgo && !appengine
// +build cgo,!appengine

package metrics

import "runtime"

func numCgoCall() int64 {
	return runtime.NumCgoCall()
}
