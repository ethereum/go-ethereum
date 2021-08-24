// +build !cgo appengine js

package metrics

func numCgoCall() int64 {
	return 0
}
