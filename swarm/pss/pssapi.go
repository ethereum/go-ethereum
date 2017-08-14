package pss

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/pot"
	"github.com/ethereum/go-ethereum/rpc"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
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
func (pssapi *API) Receive(ctx context.Context, topic whisper.TopicType) (*rpc.Subscription, error) {
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
			log.Warn(fmt.Sprintf("notification on pss sub topic rpc (sub %v) msg %v failed!", psssub.ID, msg))
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
func (pssapi *API) SendSym(topic whisper.TopicType, msg APIMsg) error {
	return pssapi.Pss.SendSym(msg.Addr, topic, msg.Msg)
}

func (pssapi *API) SendAsym(topic whisper.TopicType, msg APIMsg) error {
	return pssapi.Pss.SendAsym(msg.Addr, topic, msg.Msg)
}

// BaseAddr returns the pss node's swarm overlay address
//
// Note that the overlay address is NOT inferable. To really know the node's overlay address it must reveal it itself.
func (pssapi *API) BaseAddr() ([]byte, error) {
	return pssapi.Pss.BaseAddr(), nil
}

func (pssapi *API) GetPublicKey() []byte {
	pubkey := pssapi.PublicKey()
	return crypto.FromECDSAPub(&pubkey)
}

func (pssapi *API) SetPublicKey(addr []byte, topic whisper.TopicType, pubkey []byte) error {
	var potaddr pot.Address
	copy(potaddr[:], addr)
	pssapi.Pss.SetPeerPublicKey(potaddr, topic, crypto.ToECDSAPub(pubkey))
	return nil
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

func (self *APITest) GetSymKeys(to []byte, topic whisper.TopicType) ([]byte, error) {
	var potaddr pot.Address
	copy(potaddr[:], to)
	sendsymkey, err := self.w.GetSymKey(self.peerPool[potaddr][topic].sendsymkey)
	if err != nil {
		return nil, fmt.Errorf("get send symkey fail (peer %v topic %v): %v", to, topic, err)
	}
	recvsymkey, err := self.w.GetSymKey(self.peerPool[potaddr][topic].recvsymkey)
	if err != nil {
		return nil, fmt.Errorf("get recv symkey fail (peer %v topic %v): %v", to, topic, err)
	}
	if len(recvsymkey) != len(sendsymkey) {
		return nil, fmt.Errorf("symkey length mismatch: %d != %d", len(recvsymkey), len(sendsymkey))
	}
	returnbyte := make([]byte, len(recvsymkey)*2)
	copy(returnbyte[:len(recvsymkey)], recvsymkey)
	copy(returnbyte[len(recvsymkey):], sendsymkey)
	return returnbyte, nil
}

func (self *APITest) SendSymKey(to []byte, topic whisper.TopicType) (string, error) {
	return self.sendKey(to, &topic)
}
