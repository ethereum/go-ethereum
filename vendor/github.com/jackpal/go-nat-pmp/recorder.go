package natpmp

import "time"

type callObserver interface {
	observeCall(msg []byte, result []byte, err error)
}

// A caller that records the RPC call.
type recorder struct {
	child    caller
	observer callObserver
}

func (n *recorder) call(msg []byte, timeout time.Duration) (result []byte, err error) {
	result, err = n.child.call(msg, timeout)
	n.observer.observeCall(msg, result, err)
	return
}
