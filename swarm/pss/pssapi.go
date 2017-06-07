package pss

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/network"
)

// API is the RPC API module for Pss
type API struct {
	*Pss
}

// NewAPI constructs a PssAPI instance
func NewAPI(ps *Pss) *API {
	return &API{Pss: ps}
}

// NewMsg API endpoint creates an RPC subscription
func (pssapi *API) Receive(ctx context.Context, topic Topic) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return nil, fmt.Errorf("Subscribe not supported")
	}

	psssub := notifier.CreateSubscription()
	handler := func(msg []byte, p *p2p.Peer, from []byte) error {
		apimsg := &APIMsg{
			Msg:  msg,
			Addr: from,
		}
		if err := notifier.Notify(psssub.ID, apimsg); err != nil {
			log.Warn(fmt.Sprintf("notification on pss sub topic %v rpc (sub %v) msg %v failed!", topic, psssub.ID, msg))
		}
		return nil
	}
	deregf := pssapi.Register(&topic, handler)

	go func() {
		defer deregf()
		select {
		case err := <-psssub.Err():
			log.Warn(fmt.Sprintf("caught subscription error in pss sub topic %x: %v", topic, err))
		case <-notifier.Closed():
			log.Warn(fmt.Sprintf("rpc sub notifier closed"))
		}
	}()

	return psssub, nil
}

// SendRaw sends the message (serialized into byte slice) to a peer with topic
func (pssapi *API) Send(topic Topic, msg APIMsg) error {
	if pssapi.debug && bytes.Equal(msg.Addr, pssapi.BaseAddr()) {
		log.Warn("Pss debug enabled; send to self shortcircuit", "apimsg", msg, "topic", topic)
		env := NewEnvelope(msg.Addr, topic, msg.Msg)
		return pssapi.Process(&PssMsg{
			To:      pssapi.BaseAddr(),
			Payload: env,
		})
	}
	return pssapi.SendRaw(msg.Addr, topic, msg.Msg)
}

// PssAPITest are temporary API calls for development use only
// These symbols should not be included in production environment
type APITest struct {
	*Pss
}

// NewAPI constructs a API instance
func NewAPITest(ps *Pss) *APITest {
	return &APITest{Pss: ps}
}

// temporary for access to overlay while faking kademlia healthy routines
func (pssapitest *APITest) GetForwarder(addr []byte) (fwd struct {
	Addr  []byte
	Count int
}) {
	pssapitest.Overlay.EachConn(addr, 255, func(op network.OverlayConn, po int, isproxbin bool) bool {
		if bytes.Equal(fwd.Addr, []byte{}) {
			fwd.Addr = op.Address()
		}
		fwd.Count++
		return true
	})
	return
}

// BaseAddr gets our own overlayaddress
func (pssapitest *APITest) BaseAddr() ([]byte, error) {
	return pssapitest.Pss.BaseAddr(), nil
}
