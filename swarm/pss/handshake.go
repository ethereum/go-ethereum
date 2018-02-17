// +build !nopsshandshake

package pss

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	IsActiveHandshake = true
)

var (
	ctrlSingleton *HandshakeController
)

const (
	defaultSymKeyRequestTimeout = 1000 * 8  // max wait ms to receive a response to a handshake symkey request
	defaultSymKeyExpiryTimeout  = 1000 * 10 // ms to wait before allowing garbage collection of an expired symkey
	defaultSymKeySendLimit      = 256       // amount of messages a symkey is valid for
	defaultSymKeyCapacity       = 4         // max number of symkeys to store/send simultaneously
)

// symmetric key exchange message payload
type handshakeMsg struct {
	From    []byte
	Limit   uint16
	Keys    [][]byte
	Request uint8
	Topic   Topic
}

// internal representation of an individual symmetric key
type handshakeKey struct {
	symKeyId  *string
	pubKeyId  *string
	limit     uint16
	count     uint16
	expiredAt time.Time
}

// container for all in- and outgoing keys
// for one particular peer (public key) and topic
type handshake struct {
	outKeys []handshakeKey
	inKeys  []handshakeKey
}

// Initialization parameters for the HandshakeController
//
// SymKeyRequestExpiry: Timeout for waiting for a handshake reply
// (default 8000 ms)
//
// SymKeySendLimit: Amount of messages symmetric keys issues by
// this node is valid for (default 256)
//
// SymKeyCapacity: Ideal (and maximum) amount of symmetric keys
// held per direction per peer (default 4)
type HandshakeParams struct {
	SymKeyRequestTimeout time.Duration
	SymKeyExpiryTimeout  time.Duration
	SymKeySendLimit      uint16
	SymKeyCapacity       uint8
}

// Sane defaults for HandshakeController initialization
func NewHandshakeParams() *HandshakeParams {
	return &HandshakeParams{
		SymKeyRequestTimeout: defaultSymKeyRequestTimeout * time.Millisecond,
		SymKeyExpiryTimeout:  defaultSymKeyExpiryTimeout * time.Millisecond,
		SymKeySendLimit:      defaultSymKeySendLimit,
		SymKeyCapacity:       defaultSymKeyCapacity,
	}
}

// Singleton object enabling semi-automatic Diffie-Hellman
// exchange of ephemeral symmetric keys
type HandshakeController struct {
	pss                  *Pss
	keyC                 map[string]chan []string // adds a channel to report when a handshake succeeds
	lock                 sync.Mutex
	symKeyRequestTimeout time.Duration
	symKeyExpiryTimeout  time.Duration
	symKeySendLimit      uint16
	symKeyCapacity       uint8
	symKeyIndex          map[string]*handshakeKey
	handshakes           map[string]map[Topic]*handshake
	deregisterFuncs      map[Topic]func()
}

// Attach HandshakeController to pss node
//
// Must be called before starting the pss node service
func SetHandshakeController(pss *Pss, params *HandshakeParams) error {
	ctrl := &HandshakeController{
		pss:                  pss,
		keyC:                 make(map[string]chan []string),
		symKeyRequestTimeout: params.SymKeyRequestTimeout,
		symKeyExpiryTimeout:  params.SymKeyExpiryTimeout,
		symKeySendLimit:      params.SymKeySendLimit,
		symKeyCapacity:       params.SymKeyCapacity,
		symKeyIndex:          make(map[string]*handshakeKey),
		handshakes:           make(map[string]map[Topic]*handshake),
		deregisterFuncs:      make(map[Topic]func()),
	}
	api := &HandshakeAPI{
		namespace: "pss",
		ctrl:      ctrl,
	}
	pss.addAPI(rpc.API{
		Namespace: api.namespace,
		Version:   "0.2",
		Service:   api,
		Public:    true,
	})
	ctrlSingleton = ctrl
	return nil
}

// Return all unexpired symmetric keys from store by
// peer (public key), topic and specified direction
func (self *HandshakeController) validKeys(pubkeyid string, topic *Topic, in bool) (validkeys []*string) {
	self.lock.Lock()
	defer self.lock.Unlock()
	now := time.Now()
	if _, ok := self.handshakes[pubkeyid]; !ok {
		return []*string{}
	} else if _, ok := self.handshakes[pubkeyid][*topic]; !ok {
		return []*string{}
	}
	var keystore *[]handshakeKey
	if in {
		keystore = &(self.handshakes[pubkeyid][*topic].inKeys)
	} else {
		keystore = &(self.handshakes[pubkeyid][*topic].outKeys)
	}

	for _, key := range *keystore {
		if key.limit <= key.count {
			self.releaseKey(*key.symKeyId, topic)
		} else if !key.expiredAt.IsZero() && key.expiredAt.Before(now) {
			self.releaseKey(*key.symKeyId, topic)
		} else {
			validkeys = append(validkeys, key.symKeyId)
		}
	}
	return
}

// Add all given symmetric keys with validity limits to store by
// peer (public key), topic and specified direction
func (self *HandshakeController) updateKeys(pubkeyid string, topic *Topic, in bool, symkeyids []string, limit uint16) {
	self.lock.Lock()
	defer self.lock.Unlock()
	if _, ok := self.handshakes[pubkeyid]; !ok {
		self.handshakes[pubkeyid] = make(map[Topic]*handshake)

	}
	if self.handshakes[pubkeyid][*topic] == nil {
		self.handshakes[pubkeyid][*topic] = &handshake{}
	}
	var keystore *[]handshakeKey
	expire := time.Now()
	if in {
		keystore = &(self.handshakes[pubkeyid][*topic].inKeys)
	} else {
		keystore = &(self.handshakes[pubkeyid][*topic].outKeys)
		expire = expire.Add(time.Millisecond * self.symKeyExpiryTimeout)
	}
	for _, storekey := range *keystore {
		storekey.expiredAt = expire
	}
	for i := 0; i < len(symkeyids); i++ {
		storekey := handshakeKey{
			symKeyId: &symkeyids[i],
			pubKeyId: &pubkeyid,
			limit:    limit,
		}
		*keystore = append(*keystore, storekey)
		self.pss.symKeyPool[*storekey.symKeyId][*topic].protected = true
	}
	for i := 0; i < len(*keystore); i++ {
		self.symKeyIndex[*(*keystore)[i].symKeyId] = &((*keystore)[i])
	}
}

// Expire a symmetric key, making it elegible for garbage collection
func (self *HandshakeController) releaseKey(symkeyid string, topic *Topic) bool {
	if self.symKeyIndex[symkeyid] == nil {
		log.Debug("no symkey", "symkeyid", symkeyid)
		return false
	}
	self.symKeyIndex[symkeyid].expiredAt = time.Now()
	log.Debug("handshake release", "symkeyid", symkeyid)
	return true
}

// Checks all symmetric keys in given direction(s) by
// specified peer (public key) and topic for expiry.
// Expired means:
// - expiry timestamp is set, and grace period is exceeded
// - message validity limit is reached
func (self *HandshakeController) cleanHandshake(pubkeyid string, topic *Topic, in bool, out bool) int {
	self.lock.Lock()
	defer self.lock.Unlock()
	var deletecount int
	var deletes []string
	now := time.Now()
	handshake := self.handshakes[pubkeyid][*topic]
	log.Debug("handshake clean", "pubkey", pubkeyid, "topic", topic)
	if in {
		for i, key := range handshake.inKeys {
			if key.expiredAt.Before(now) || (key.expiredAt.IsZero() && key.limit <= key.count) {
				log.Trace("handshake in clean remove", "symkeyid", *key.symKeyId)
				deletes = append(deletes, *key.symKeyId)
				handshake.inKeys[deletecount] = handshake.inKeys[i]
				deletecount++
			}
		}
		handshake.inKeys = handshake.inKeys[:len(handshake.inKeys)-deletecount]
	}
	if out {
		deletecount = 0
		for i, key := range handshake.outKeys {
			if key.expiredAt.Before(now) && (key.expiredAt.IsZero() && key.limit <= key.count) {
				log.Trace("handshake out clean remove", "symkeyid", *key.symKeyId)
				deletes = append(deletes, *key.symKeyId)
				handshake.outKeys[deletecount] = handshake.outKeys[i]
				deletecount++
			}
		}
		handshake.outKeys = handshake.outKeys[:len(handshake.outKeys)-deletecount]
	}
	for _, keyid := range deletes {
		delete(self.symKeyIndex, keyid)
		self.pss.symKeyPool[keyid][*topic].protected = false
	}
	return len(deletes)
}

// Runs cleanHandshake() on all peers and topics
func (self *HandshakeController) clean() {
	peerpubkeys := self.handshakes
	for pubkeyid, peertopics := range peerpubkeys {
		for topic := range peertopics {
			self.cleanHandshake(pubkeyid, &topic, true, true)
		}
	}
}

// Passed as a PssMsg handler for the topic handshake is activated on
// Handles incoming key exchange messages and
// ccunts message usage by symmetric key (expiry limit control)
// Only returns error if key handler fails
func (self *HandshakeController) handler(msg []byte, p *p2p.Peer, asymmetric bool, symkeyid string) error {
	if !asymmetric {
		if self.symKeyIndex[symkeyid] != nil {
			if self.symKeyIndex[symkeyid].count >= self.symKeyIndex[symkeyid].limit {
				return fmt.Errorf("discarding message using expired key: %s", symkeyid)
			}
			self.symKeyIndex[symkeyid].count++
			log.Trace("increment symkey recv use", "symsymkeyid", symkeyid, "count", self.symKeyIndex[symkeyid].count, "limit", self.symKeyIndex[symkeyid].limit, "receiver", common.ToHex(crypto.FromECDSAPub(self.pss.PublicKey())))
		}
		return nil
	}
	keymsg := &handshakeMsg{}
	err := rlp.DecodeBytes(msg, keymsg)
	if err == nil {
		err := self.handleKeys(symkeyid, keymsg)
		if err != nil {
			log.Error("handlekeys fail", "error", err)
		}
		return err
	}
	return nil
}

// Handle incoming key exchange message
// Add keys received from peer to store
// and enerate and send the amount of keys requested by peer
//
// TODO:
// - flood guard
// - keylength check
// - update address hint if:
//   1) leftmost bytes in new address do not match stored
//   2) else, if new address is longer
func (self *HandshakeController) handleKeys(pubkeyid string, keymsg *handshakeMsg) error {
	// new keys from peer
	if len(keymsg.Keys) > 0 {
		log.Debug("received handshake keys", "pubkeyid", pubkeyid, "from", keymsg.From, "count", len(keymsg.Keys))
		var sendsymkeyids []string
		for _, key := range keymsg.Keys {
			sendsymkey := make([]byte, len(key))
			copy(sendsymkey, key)
			var address PssAddress
			copy(address[:], keymsg.From)
			sendsymkeyid, err := self.pss.SetSymmetricKey(sendsymkey, keymsg.Topic, &address, false)
			if err != nil {
				return err
			}
			sendsymkeyids = append(sendsymkeyids, sendsymkeyid)
		}
		if len(sendsymkeyids) > 0 {
			self.updateKeys(pubkeyid, &keymsg.Topic, false, sendsymkeyids, keymsg.Limit)

			self.alertHandshake(pubkeyid, sendsymkeyids)
		}
	}

	// peer request for keys
	if keymsg.Request > 0 {
		_, err := self.sendKey(pubkeyid, &keymsg.Topic, keymsg.Request)
		if err != nil {
			return err
		}
	}

	return nil
}

// Send key exchange to peer (public key) valid for `topic`
// Will send number of keys specified by `keycount` with
// validity limits specified in `msglimit`
// If number of valid outgoing keys is less than the ideal/max
// amount, a request is sent for the amount of keys to make up
// the difference
func (self *HandshakeController) sendKey(pubkeyid string, topic *Topic, keycount uint8) ([]string, error) {

	var requestcount uint8
	to := &PssAddress{}
	if _, ok := self.pss.pubKeyPool[pubkeyid]; !ok {
		return []string{}, errors.New("Invalid public key")
	} else if psp, ok := self.pss.pubKeyPool[pubkeyid][*topic]; ok {
		to = psp.address
	}

	recvkeys := make([][]byte, keycount)
	recvkeyids := make([]string, keycount)
	self.lock.Lock()
	if _, ok := self.handshakes[pubkeyid]; !ok {
		self.handshakes[pubkeyid] = make(map[Topic]*handshake)
	}
	self.lock.Unlock()

	// check if buffer is not full
	outkeys := self.validKeys(pubkeyid, topic, false)
	if len(outkeys) < int(self.symKeyCapacity) {
		//requestcount = uint8(self.symKeyCapacity - uint8(len(outkeys)))
		requestcount = self.symKeyCapacity
	}
	// return if there's nothing to be accomplished
	if requestcount == 0 && keycount == 0 {
		return []string{}, nil
	}

	// generate new keys to send
	for i := 0; i < len(recvkeyids); i++ {
		var err error
		recvkeyids[i], err = self.pss.generateSymmetricKey(*topic, to, true)
		if err != nil {
			return []string{}, fmt.Errorf("set receive symkey fail (pubkey %x topic %x): %v", pubkeyid, topic, err)
		}
		recvkeys[i], err = self.pss.GetSymmetricKey(recvkeyids[i])
		if err != nil {
			return []string{}, fmt.Errorf("GET Generated outgoing symkey fail (pubkey %x topic %x): %v", pubkeyid, topic, err)
		}
	}
	self.updateKeys(pubkeyid, topic, true, recvkeyids, self.symKeySendLimit)

	// encode and send the message
	recvkeymsg := &handshakeMsg{
		From:    self.pss.BaseAddr(),
		Keys:    recvkeys,
		Request: requestcount,
		Limit:   self.symKeySendLimit,
		Topic:   *topic,
	}
	log.Debug("sending our symkeys", "pubkey", pubkeyid, "symkeys", recvkeyids, "limit", self.symKeySendLimit, "requestcount", requestcount, "keycount", len(recvkeys))
	recvkeybytes, err := rlp.EncodeToBytes(recvkeymsg)
	if err != nil {
		return []string{}, fmt.Errorf("rlp keymsg encode fail: %v", err)
	}
	// if the send fails it means this public key is not registered for this particular address AND topic
	err = self.pss.SendAsym(pubkeyid, *topic, recvkeybytes)
	if err != nil {
		return []string{}, fmt.Errorf("Send symkey failed: %v", err)
	}
	return recvkeyids, nil
}

// Enables callback for keys received from a key exchange request
func (self *HandshakeController) alertHandshake(pubkeyid string, symkeys []string) chan []string {
	if len(symkeys) > 0 {
		if _, ok := self.keyC[pubkeyid]; ok {
			self.keyC[pubkeyid] <- symkeys
			close(self.keyC[pubkeyid])
			delete(self.keyC, pubkeyid)
		}
		return nil
	} else {
		if _, ok := self.keyC[pubkeyid]; !ok {
			self.keyC[pubkeyid] = make(chan []string)
		}
	}
	return self.keyC[pubkeyid]
}

type HandshakeAPI struct {
	namespace string
	ctrl      *HandshakeController
}

// Initiate a handshake session for a peer (public key) and topic
// combination.
//
// If `sync` is set, the call will block until keys are received from peer,
// or if the handshake request times out
//
// If `flush` is set, the max amount of keys will be sent to the peer
// regardless of how many valid keys that currently exist in the store.
//
// Returns list of symmetric key ids that can be passed to pss.GetSymmetricKey()
// for retrieval of the symmetric key bytes themselves.
//
// Fails if the incoming symmetric key store is already full (and `flush` is false),
// or if the underlying key dispatcher fails
func (self *HandshakeAPI) Handshake(pubkeyid string, topic Topic, sync bool, flush bool) (keys []string, err error) {
	var hsc chan []string
	var keycount uint8
	if flush {
		keycount = self.ctrl.symKeyCapacity
	} else {
		validkeys := self.ctrl.validKeys(pubkeyid, &topic, false)
		keycount = self.ctrl.symKeyCapacity - uint8(len(validkeys))
	}
	if keycount == 0 {
		return keys, errors.New("Incoming symmetric key store is already full")
	}
	if sync {
		hsc = self.ctrl.alertHandshake(pubkeyid, []string{})
	}
	_, err = self.ctrl.sendKey(pubkeyid, &topic, keycount)
	if err != nil {
		return keys, err
	}
	if sync {
		ctx, cancel := context.WithTimeout(context.Background(), self.ctrl.symKeyRequestTimeout)
		defer cancel()
		select {
		case keys = <-hsc:
			log.Trace("sync handshake response receive", "key", keys)
		case <-ctx.Done():
			return []string{}, errors.New("timeout")
		}
	}
	return keys, nil
}

// Activate handshake functionality on a topic
func (self *HandshakeAPI) AddHandshake(topic Topic) error {
	self.ctrl.deregisterFuncs[topic] = self.ctrl.pss.Register(&topic, self.ctrl.handler)
	return nil
}

// Deactivate handshake functionality on a topic
func (self *HandshakeAPI) RemoveHandshake(topic *Topic) error {
	if _, ok := self.ctrl.deregisterFuncs[*topic]; ok {
		self.ctrl.deregisterFuncs[*topic]()
	}
	return nil
}

// Returns all valid symmetric keys in store per peer (public key)
// and topic.
//
// The `in` and `out` parameters indicate for which direction(s)
// symmetric keys will be returned.
// If both are false, no keys (and no error) will be returned.
func (self *HandshakeAPI) GetHandshakeKeys(pubkeyid string, topic Topic, in bool, out bool) (keys []string, err error) {
	if in {
		for _, inkey := range self.ctrl.validKeys(pubkeyid, &topic, true) {
			keys = append(keys, *inkey)
		}
	}
	if out {
		for _, outkey := range self.ctrl.validKeys(pubkeyid, &topic, false) {
			keys = append(keys, *outkey)
		}
	}
	return keys, nil
}

// Returns the amount of messages the specified symmetric key
// is still valid for under the handshake scheme
func (self *HandshakeAPI) GetHandshakeKeyCapacity(symkeyid string) (uint16, error) {
	storekey := self.ctrl.symKeyIndex[symkeyid]
	if storekey == nil {
		return 0, fmt.Errorf("invalid symkey id %s", symkeyid)
	}
	return storekey.limit - storekey.count, nil
}

// Returns the byte representation of the public key in ascii hex
// associated with the given symmetric key
func (self *HandshakeAPI) GetHandshakePublicKey(symkeyid string) (string, error) {
	storekey := self.ctrl.symKeyIndex[symkeyid]
	if storekey == nil {
		return "", fmt.Errorf("invalid symkey id %s", symkeyid)
	}
	return *storekey.pubKeyId, nil
}

// Manually expire the given symkey
//
// If `flush` is set, garbage collection will be performed before returning.
//
// Returns true on successful removal, false otherwise
func (self *HandshakeAPI) ReleaseHandshakeKey(pubkeyid string, topic Topic, symkeyid string, flush bool) (removed bool, err error) {
	removed = self.ctrl.releaseKey(symkeyid, &topic)
	if removed && flush {
		self.ctrl.cleanHandshake(pubkeyid, &topic, true, true)
	}
	return
}

// Send symmetric message under the handshake scheme
//
// Overloads the pss.SendSym() API call, adding symmetric key usage count
// for message expiry control
func (self *HandshakeAPI) SendSym(symkeyid string, topic Topic, msg hexutil.Bytes) (err error) {
	err = self.ctrl.pss.SendSym(symkeyid, topic, msg[:])
	if self.ctrl.symKeyIndex[symkeyid] != nil {
		if self.ctrl.symKeyIndex[symkeyid].count >= self.ctrl.symKeyIndex[symkeyid].limit {
			return errors.New("attempted send with expired key")
		}
		self.ctrl.symKeyIndex[symkeyid].count++
		log.Trace("increment symkey send use", "symkeyid", symkeyid, "count", self.ctrl.symKeyIndex[symkeyid].count, "limit", self.ctrl.symKeyIndex[symkeyid].limit, "receiver", common.ToHex(crypto.FromECDSAPub(self.ctrl.pss.PublicKey())))
	}
	return
}
