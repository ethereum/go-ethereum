package pss

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
)

// PssAPI is the RPC API module for Pss
type PssAPI struct {
	*Pss
}

// NewPssAPI constructs a PssAPI instance
func NewPssAPI(ps *Pss) *PssAPI {
	return &PssAPI{Pss: ps}
}

// PssAPIMsg is the type for messages, it extends the rlp encoded protocol Msg
// with the Sender's overlay address
type PssAPIMsg struct {
	Msg  []byte
	Addr []byte
}

// NewMsg API endpoint creates an RPC subscription
func (pssapi *PssAPI) NewMsg(ctx context.Context, topic PssTopic) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return nil, fmt.Errorf("Subscribe not supported")
	}

	psssub := notifier.CreateSubscription()
	handler := func(msg []byte, p *p2p.Peer, from []byte) error {
		apimsg := &PssAPIMsg{
			Msg: msg,
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
		//defer psssub.Unsubscribe()
		select {
		case err := <-psssub.Err():
			log.Warn(fmt.Sprintf("caught subscription error in pss sub topic: %v", topic, err))
		case <-notifier.Closed():
			log.Warn(fmt.Sprintf("rpc sub notifier closed"))
		}
	}()

	return psssub, nil
}

// SendRaw sends the message (serialised into byte slice) to a peer with topic
func (pssapi *PssAPI) SendRaw(topic PssTopic, msg PssAPIMsg) error {
	err := pssapi.Pss.Send(msg.Addr, topic, msg.Msg)
	if err != nil {
		return fmt.Errorf("send error: %v", err)
	}
	return fmt.Errorf("ok sent")
}

// BaseAddr gets our own overlayaddress
func (pssapi *PssAPI) BaseAddr() ([]byte, error) { 
	log.Warn("inside baseaddr")
	return pssapi.Pss.Overlay.BaseAddr(), nil
}
