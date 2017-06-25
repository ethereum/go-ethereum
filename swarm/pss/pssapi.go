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

// Pss API services
type API struct {
	*Pss
}

func NewAPI(ps *Pss) *API {
	return &API{Pss: ps}
}

// Creates a new subscription for the caller. Enables external handling of incoming messages.
//
// A new handler is registered in pss for the supplied topic
//
// All incoming messages to the node matching this topic will be encapsulated in the APIMsg struct and sent to the subscriber
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

// Sends the message wrapped in APIMsg through pss
//
// Wrapper method for the pss.SendRaw function.
//
// The method will pass on the error received from pss.
//
// Note that normally pss will report an error if an attempt is made to send a pss to oneself. However, if the debug flag has been set, and the address specified in APIMsg is the node's own, this method implements a short-circuit which injects the message as an incoming message (using Pss.Process). This can be useful for testing purposes, when only operating with one node.
func (pssapi *API) Send(topic Topic, msg APIMsg) error {
	if pssapi.debug && bytes.Equal(msg.Addr, pssapi.Pss.BaseAddr()) {
		log.Warn("Pss debug enabled; send to self shortcircuit", "apimsg", msg, "topic", topic)
		env := NewEnvelope(msg.Addr, topic, msg.Msg)
		return pssapi.Process(&PssMsg{
			To:      pssapi.Pss.BaseAddr(),
			Payload: env,
		})
	}
	return pssapi.SendRaw(msg.Addr, topic, msg.Msg)
}

// BaseAddr returns the pss node's swarm overlay address
//
// Note that the overlay address is NOT inferable. To really know the node's overlay address it must reveal it itself.
func (pssapi *API) BaseAddr() ([]byte, error) {
	return pssapi.Pss.BaseAddr(), nil
}

// PssAPITest are temporary API calls for development use only
//
// These symbols should NOT be included in production environment
type APITest struct {
	*Pss
}

// Include these methods to the node.Service if test symbols should be used
func NewAPITest(ps *Pss) *APITest {
	return &APITest{Pss: ps}
}

// Get the current nearest swarm node to the specified address
//
// (Can be used for diagnosing kademlia state)
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
