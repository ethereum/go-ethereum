//go:build cgo && !appengine && !js
// +build cgo,!appengine,!js

package metrics

import "runtime"

func numCgoCall() int64 {
	return runtime.NumCgoCall()
}
