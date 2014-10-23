package p2p

import (
	// "fmt"
	"net"
	"testing"
	"time"
)

func TestPeerErrorHandler(t *testing.T) {
	address := &net.TCPAddr{IP: net.IP([]byte{1, 2, 3, 4}), Port: 30303}
	peerDisconnect := make(chan DisconnectRequest)
	peerErrorChan := NewPeerErrorChannel()
	peh := NewPeerErrorHandler(address, peerDisconnect, peerErrorChan, NewBlacklist())
	peh.Start()
	defer peh.Stop()
	for i := 0; i < 11; i++ {
		select {
		case <-peerDisconnect:
			t.Errorf("expected no disconnect request")
		default:
		}
		peerErrorChan <- NewPeerError(MiscError, "")
	}
	time.Sleep(1 * time.Millisecond)
	select {
	case request := <-peerDisconnect:
		if request.addr.String() != address.String() {
			t.Errorf("incorrect address %v != %v", request.addr, address)
		}
	default:
		t.Errorf("expected disconnect request")
	}
}
