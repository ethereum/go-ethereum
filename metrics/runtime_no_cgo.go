//go:build !cgo || appengine
// +build !cgo appengine

package metrics

func numCgoCall() int64 {
	return 0
}
