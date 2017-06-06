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

// PssAPI is the RPC API module for Pss
type PssAPI struct {
	*Pss
}

// NewPssAPI constructs a PssAPI instance
func NewPssAPI(ps *Pss) *PssAPI {
	return &PssAPI{Pss: ps}
}

// NewMsg API endpoint creates an RPC subscription
func (pssapi *PssAPI) ReceivePss(ctx context.Context, topic PssTopic) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return nil, fmt.Errorf("Subscribe not supported")
	}

	psssub := notifier.CreateSubscription()
	handler := func(msg []byte, p *p2p.Peer, from []byte) error {
		apimsg := &PssAPIMsg{
			Msg:  msg,
			Addr: from,
		}
		if err := notifier.Notify(psssub.ID, apimsg); err != nil {
			log.Warn(fmt.Sprintf("notification on pss sub topic %v rpc (sub %v) msg %v failed!", topic, psssub.ID, msg))
		}
		return nil
	}
	deregf := pssapi.Pss.Register(&topic, handler)

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
func (pssapi *PssAPI) SendPss(topic PssTopic, msg PssAPIMsg) error {
	if pssapi.Pss.debug && bytes.Equal(msg.Addr, pssapi.Pss.BaseAddr()) {
		log.Warn("Pss debug enabled; send to self shortcircuit", "apimsg", msg, "topic", topic)
		env := NewPssEnvelope(msg.Addr, topic, msg.Msg)
		return pssapi.Pss.Process(&PssMsg{
			To: pssapi.Pss.BaseAddr(),
			Payload: env,
		})
	}
	return pssapi.Pss.Send(msg.Addr, topic, msg.Msg)
	/*if err != nil {
		return fmt.Errorf("send error: %v", err)
	}
	return fmt.Errorf("ok sent")*/
}

// PssAPITest are temporary API calls for development use only
// These symbols should not be included in production environment
type PssAPITest struct {
	*Pss
}

// NewPssAPI constructs a PssAPI instance
func NewPssAPITest(ps *Pss) *PssAPITest {
	return &PssAPITest{Pss: ps}
}

// temporary for access to overlay while faking kademlia healthy routines
func (pssapitest *PssAPITest) GetForwarder(addr []byte) (fwd struct {
	Addr  []byte
	Count int
}) {
	pssapitest.Pss.Overlay.EachConn(addr, 255, func(op network.OverlayConn, po int, isproxbin bool) bool {
		if bytes.Equal(fwd.Addr, []byte{}) {
			fwd.Addr = op.Address()
		}
		fwd.Count++
		return true
	})
	return
}

// BaseAddr gets our own overlayaddress
func (pssapitest *PssAPITest) BaseAddr() ([]byte, error) {
	return pssapitest.Pss.BaseAddr(), nil
}
