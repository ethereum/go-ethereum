package notify

import (
	"crypto/ecdsa"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/pss"
)

const (
	// sent from requester to updater to request start of notifications
	MsgCodeStart = iota

	// sent from updater to requester, contains a notification plus a new symkey to replace the old
	MsgCodeNotifyWithKey

	// sent from updater to requester, contains a notification
	MsgCodeNotify

	// sent from requester to updater to request stop of notifications (currently unused)
	MsgCodeStop
	MsgCodeMax
)

const (
	DefaultAddressLength = 1
	symKeyLength         = 32 // this should be gotten from source
)

var (
	// control topic is used before symmetric key issuance completes
	controlTopic = pss.Topic{0x00, 0x00, 0x00, 0x01}
)

// when code is MsgCodeStart, Payload is address
// when code is MsgCodeNotifyWithKey, Payload is notification | symkey
// when code is MsgCodeNotify, Payload is notification
// when code is MsgCodeStop, Payload is address
type Msg struct {
	Code       byte
	Name       []byte
	Payload    []byte
	namestring string
}

// NewMsg creates a new notification message object
func NewMsg(code byte, name string, payload []byte) *Msg {
	return &Msg{
		Code:       code,
		Name:       []byte(name),
		Payload:    payload,
		namestring: name,
	}
}

// NewMsgFromPayload decodes a serialized message payload into a new notification message object
func NewMsgFromPayload(payload []byte) (*Msg, error) {
	msg := &Msg{}
	err := rlp.DecodeBytes(payload, msg)
	if err != nil {
		return nil, err
	}
	msg.namestring = string(msg.Name)
	return msg, nil
}

// a notifier has one sendBin entry for each address space it sends messages to
type sendBin struct {
	address  pss.PssAddress
	symKeyId string
	count    int
}

// represents a single notification service
// only subscription address bins that match the address of a notification client have entries.
type notifier struct {
	bins      map[string]*sendBin
	topic     pss.Topic // identifies the resource for pss receiver
	threshold int       // amount of address bytes used in bins
	updateC   <-chan []byte
	quitC     chan struct{}
}

func (n *notifier) removeSubscription() {
	n.quitC <- struct{}{}
}

// represents an individual subscription made by a public key at a specific address/neighborhood
type subscription struct {
	pubkeyId string
	address  pss.PssAddress
	handler  func(string, []byte) error
}

// Controller is the interface to control, add and remove notification services and subscriptions
type Controller struct {
	pss           *pss.Pss
	notifiers     map[string]*notifier
	subscriptions map[string]*subscription
	mu            sync.Mutex
}

// NewController creates a new Controller object
func NewController(ps *pss.Pss) *Controller {
	ctrl := &Controller{
		pss:           ps,
		notifiers:     make(map[string]*notifier),
		subscriptions: make(map[string]*subscription),
	}
	ctrl.pss.Register(&controlTopic, pss.NewHandler(ctrl.Handler))
	return ctrl
}

// IsActive is used to check if a notification service exists for a specified id string
// Returns true if exists, false if not
func (c *Controller) IsActive(name string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.isActive(name)
}

func (c *Controller) isActive(name string) bool {
	_, ok := c.notifiers[name]
	return ok
}

// Subscribe is used by a client to request notifications from a notification service provider
// It will create a MsgCodeStart message and send asymmetrically to the provider using its public key and routing address
// The handler function is a callback that will be called when notifications are received
// Fails if the request pss cannot be sent or if the update message could not be serialized
func (c *Controller) Subscribe(name string, pubkey *ecdsa.PublicKey, address pss.PssAddress, handler func(string, []byte) error) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	msg := NewMsg(MsgCodeStart, name, c.pss.BaseAddr())
	c.pss.SetPeerPublicKey(pubkey, controlTopic, &address)
	pubkeyId := hexutil.Encode(crypto.FromECDSAPub(pubkey))
	smsg, err := rlp.EncodeToBytes(msg)
	if err != nil {
		return err
	}
	err = c.pss.SendAsym(pubkeyId, controlTopic, smsg)
	if err != nil {
		return err
	}
	c.subscriptions[name] = &subscription{
		pubkeyId: pubkeyId,
		address:  address,
		handler:  handler,
	}
	return nil
}

// Unsubscribe, perhaps unsurprisingly, undoes the effects of Subscribe
// Fails if the subscription does not exist, if the request pss cannot be sent or if the update message could not be serialized
func (c *Controller) Unsubscribe(name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	sub, ok := c.subscriptions[name]
	if !ok {
		return fmt.Errorf("Unknown subscription '%s'", name)
	}
	msg := NewMsg(MsgCodeStop, name, sub.address)
	smsg, err := rlp.EncodeToBytes(msg)
	if err != nil {
		return err
	}
	err = c.pss.SendAsym(sub.pubkeyId, controlTopic, smsg)
	if err != nil {
		return err
	}
	delete(c.subscriptions, name)
	return nil
}

// NewNotifier is used by a notification service provider to create a new notification service
// It takes a name as identifier for the resource, a threshold indicating the granularity of the subscription address bin
// It then starts an event loop which listens to the supplied update channel and executes notifications on channel receives
// Fails if a notifier already is registered on the name
//func (c *Controller) NewNotifier(name string, threshold int, contentFunc func(string) ([]byte, error)) error {
func (c *Controller) NewNotifier(name string, threshold int, updateC <-chan []byte) (func(), error) {
	c.mu.Lock()
	if c.isActive(name) {
		c.mu.Unlock()
		return nil, fmt.Errorf("Notification service %s already exists in controller", name)
	}
	quitC := make(chan struct{})
	c.notifiers[name] = &notifier{
		bins:      make(map[string]*sendBin),
		topic:     pss.BytesToTopic([]byte(name)),
		threshold: threshold,
		updateC:   updateC,
		quitC:     quitC,
		//contentFunc: contentFunc,
	}
	c.mu.Unlock()
	go func() {
		for {
			select {
			case <-quitC:
				return
			case data := <-updateC:
				c.notify(name, data)
			}
		}
	}()

	return c.notifiers[name].removeSubscription, nil
}

// RemoveNotifier is used to stop a notification service.
// It cancels the event loop listening to the notification provider's update channel
func (c *Controller) RemoveNotifier(name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	currentNotifier, ok := c.notifiers[name]
	if !ok {
		return fmt.Errorf("Unknown notification service %s", name)
	}
	currentNotifier.removeSubscription()
	delete(c.notifiers, name)
	return nil
}

// Notify is called by a notification service provider to issue a new notification
// It takes the name of the notification service and the data to be sent.
// It fails if a notifier with this name does not exist or if data could not be serialized
// Note that it does NOT fail on failure to send a message
func (c *Controller) notify(name string, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.isActive(name) {
		return fmt.Errorf("Notification service %s doesn't exist", name)
	}
	msg := NewMsg(MsgCodeNotify, name, data)
	smsg, err := rlp.EncodeToBytes(msg)
	if err != nil {
		return err
	}
	for _, m := range c.notifiers[name].bins {
		log.Debug("sending pss notify", "name", name, "addr", fmt.Sprintf("%x", m.address), "topic", fmt.Sprintf("%x", c.notifiers[name].topic), "data", data)
		go func(m *sendBin) {
			err = c.pss.SendSym(m.symKeyId, c.notifiers[name].topic, smsg)
			if err != nil {
				log.Warn("Failed to send notify to addr %x: %v", m.address, err)
			}
		}(m)
	}
	return nil
}

// check if we already have the bin
// if we do, retrieve the symkey from it and increment the count
// if we dont make a new symkey and a new bin entry
func (c *Controller) addToBin(ntfr *notifier, address []byte) (symKeyId string, pssAddress pss.PssAddress, err error) {

	// parse the address from the message and truncate if longer than our bins threshold
	if len(address) > ntfr.threshold {
		address = address[:ntfr.threshold]
	}

	pssAddress = pss.PssAddress(address)
	hexAddress := fmt.Sprintf("%x", address)
	currentBin, ok := ntfr.bins[hexAddress]
	if ok {
		currentBin.count++
		symKeyId = currentBin.symKeyId
	} else {
		symKeyId, err = c.pss.GenerateSymmetricKey(ntfr.topic, &pssAddress, false)
		if err != nil {
			return "", nil, err
		}
		ntfr.bins[hexAddress] = &sendBin{
			address:  address,
			symKeyId: symKeyId,
			count:    1,
		}
	}
	return symKeyId, pssAddress, nil
}

func (c *Controller) handleStartMsg(msg *Msg, keyid string) (err error) {

	keyidbytes, err := hexutil.Decode(keyid)
	if err != nil {
		return err
	}
	pubkey, err := crypto.UnmarshalPubkey(keyidbytes)
	if err != nil {
		return err
	}

	// if name is not registered for notifications we will not react
	currentNotifier, ok := c.notifiers[msg.namestring]
	if !ok {
		return fmt.Errorf("Subscribe attempted on unknown resource '%s'", msg.namestring)
	}

	// add to or open new bin
	symKeyId, pssAddress, err := c.addToBin(currentNotifier, msg.Payload)
	if err != nil {
		return err
	}

	// add to address book for send initial notify
	symkey, err := c.pss.GetSymmetricKey(symKeyId)
	if err != nil {
		return err
	}
	err = c.pss.SetPeerPublicKey(pubkey, controlTopic, &pssAddress)
	if err != nil {
		return err
	}

	// TODO this is set to zero-length byte pending decision on protocol for initial message, whether it should include message or not, and how to trigger the initial message so that current state of Swarm feed is sent upon subscription
	notify := []byte{}
	replyMsg := NewMsg(MsgCodeNotifyWithKey, msg.namestring, make([]byte, len(notify)+symKeyLength))
	copy(replyMsg.Payload, notify)
	copy(replyMsg.Payload[len(notify):], symkey)
	sReplyMsg, err := rlp.EncodeToBytes(replyMsg)
	if err != nil {
		return err
	}
	return c.pss.SendAsym(keyid, controlTopic, sReplyMsg)
}

func (c *Controller) handleNotifyWithKeyMsg(msg *Msg) error {
	symkey := msg.Payload[len(msg.Payload)-symKeyLength:]
	topic := pss.BytesToTopic(msg.Name)

	// \TODO keep track of and add actual address
	updaterAddr := pss.PssAddress([]byte{})
	c.pss.SetSymmetricKey(symkey, topic, &updaterAddr, true)
	c.pss.Register(&topic, pss.NewHandler(c.Handler))
	return c.subscriptions[msg.namestring].handler(msg.namestring, msg.Payload[:len(msg.Payload)-symKeyLength])
}

func (c *Controller) handleStopMsg(msg *Msg) error {
	// if name is not registered for notifications we will not react
	currentNotifier, ok := c.notifiers[msg.namestring]
	if !ok {
		return fmt.Errorf("Unsubscribe attempted on unknown resource '%s'", msg.namestring)
	}

	// parse the address from the message and truncate if longer than our bins' address length threshold
	address := msg.Payload
	if len(msg.Payload) > currentNotifier.threshold {
		address = address[:currentNotifier.threshold]
	}

	// remove the entry from the bin if it exists, and remove the bin if it's the last remaining one
	hexAddress := fmt.Sprintf("%x", address)
	currentBin, ok := currentNotifier.bins[hexAddress]
	if !ok {
		return fmt.Errorf("found no active bin for address %s", hexAddress)
	}
	currentBin.count--
	if currentBin.count == 0 { // if no more clients in this bin, remove it
		delete(currentNotifier.bins, hexAddress)
	}
	return nil
}

// Handler is the pss topic handler to be used to process notification service messages
// It should be registered in the pss of both to any notification service provides and clients using the service
func (c *Controller) Handler(smsg []byte, p *p2p.Peer, asymmetric bool, keyid string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	log.Debug("notify controller handler", "keyid", keyid)

	// see if the message is valid
	msg, err := NewMsgFromPayload(smsg)
	if err != nil {
		return err
	}

	switch msg.Code {
	case MsgCodeStart:
		return c.handleStartMsg(msg, keyid)
	case MsgCodeNotifyWithKey:
		return c.handleNotifyWithKeyMsg(msg)
	case MsgCodeNotify:
		return c.subscriptions[msg.namestring].handler(msg.namestring, msg.Payload)
	case MsgCodeStop:
		return c.handleStopMsg(msg)
	}

	return fmt.Errorf("Invalid message code: %d", msg.Code)
}
