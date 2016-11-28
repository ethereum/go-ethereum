/*
Package protocols is an extension to p2p. It offers a user friendly simple way to define
devp2p subprotocols by abstracting away code standardly shared by protocols.
The subprotocol architecture is inspired by the node package. Similar to a node
the standard protocol (class) registers service contructors that are instantiated as service
instances on the protocol isntance that is launched on a p2p peer connection.

By mounting various protocol modules protocols can encapsulate vertical slices of business logic
without duplicating code related to protocol communication.
Standard protocol supports:

* mounting services instantiated with the remote peer when a protocol instance is launched on a newly
  established peer connection
* registering module-specific handshakes and offers validation and renegotiation  of handshakes
* registering multiple handlers for incoming messages
* automate assigments of code indexes to messages
* automate RLP decoding/encoding based on reflecting
* provide the forever loop to read incoming messages
* standardise error handling related to communication
* enables access to sister services of the same peer connection analogous to node.Service
* TODO: automatic generation of wire protocol specification for peers
* peerPool abstracting out peer management by defining a peerPool that is called to register/unregister
  peers as they connect and drop (ideally the peerPool also implements the peerPool interface that the
  p2p server needs to suggest peers to connect to in server-as-initiator mode of operation
  see https://github.com/ethereum/go-ethereum/issues/2254 for the peer management/connectivity related
  aspect

*/

package protocols

import (
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p"
)

// error codes used by this  protocol scheme
const (
	ErrMsgTooLong = iota
	ErrDecode
	ErrWrite
	ErrInvalidMsgCode
	ErrInvalidMsgType
	ErrLocalHandshake
	ErrRemoteHandshake
	ErrNoHandler
	ErrHandler
)

// error description strings associated with the codes
var errorToString = map[int]string{
	ErrMsgTooLong:      "Message too long",
	ErrDecode:          "Invalid message (RLP error)",
	ErrWrite:           "Error sending message",
	ErrInvalidMsgCode:  "Invalid message code",
	ErrInvalidMsgType:  "Invalid message type",
	ErrLocalHandshake:  "Local handshake error",
	ErrRemoteHandshake: "Remote handshake error",
	ErrNoHandler:       "No handler registered error",
	ErrHandler:         "Message handler error",
}

/*
Error implements the standard go error interface.
Use:

  errorf(code, format, params ...interface{})

Prints as:

 <description>: <details>

where description is given by code in errorToString
and details is fmt.Sprintf(format, params...)

exported field Code can be checked
*/
type Error struct {
	Code    int
	message string
	format  string
	params  []interface{}
}

func (self Error) Error() (message string) {
	if len(message) == 0 {
		name, ok := errorToString[self.Code]
		if !ok {
			panic("invalid message code")
		}
		self.message = name
		if self.format != "" {
			self.message += ": " + fmt.Sprintf(self.format, self.params...)
		}
	}
	return self.message
}

func errorf(code int, format string, params ...interface{}) *Error {
	self := &Error{
		Code:   code,
		format: format,
		params: params,
	}

	return self
}

// implements the code table spec
// listing the message codes and types etc
// and further metadata about the protocol
type CodeMap struct {
	Name       string                // name of the protocol
	Version    uint                  // version
	MaxMsgSize int                   // max length of message payload size
	codes      []reflect.Type        // index of codes to msg types - to create zero values
	messages   map[reflect.Type]uint // index of types to codes, for sending by type
}

func NewCodeMap(name string, version uint, maxMsgSize int, msgs ...interface{}) *CodeMap {
	self := &CodeMap{
		Name:       name,
		Version:    version,
		MaxMsgSize: maxMsgSize,
		messages:   make(map[reflect.Type]uint),
	}
	self.Register(msgs...)
	return self
}

func (self *CodeMap) Length() uint64 {
	return uint64(len(self.codes))
}

func (self *CodeMap) Register(msgs ...interface{}) {
	code := uint(len(self.codes))
	for _, msg := range msgs {
		typ := reflect.TypeOf(msg)
		_, found := self.messages[typ]
		if found {
			// ignore duplicates
			continue
		}
		// next code assigned to message type typ
		self.messages[typ] = code
		self.codes = append(self.codes, typ)
		code++
	}
}

// A Peer represents a remote peer or protocol instance that is running on a peer connection with
// a remote peer
type Peer struct {
	ct         *CodeMap                                   // CodeMap for the protocol
	m          Messenger                                  // defines senf and receive
	*p2p.Peer                                             // the p2p.Peer object representing the remote
	rw         p2p.MsgReadWriter                          // p2p.MsgReadWriter to send messages to and read messages from
	handlers   map[reflect.Type][]func(interface{}) error //  message type -> message handler callback(s) map
	disconnect func()                                     // Disconnect function set differently for testing
}

type Messenger interface {
	SendMsg(p2p.MsgWriter, uint64, interface{}) error
	ReadMsg(p2p.MsgReader) (p2p.Msg, error)
}

// NewPeer returns a new peer
// this constructor is called by the p2p.Protocol#Run function
// the first two arguments are comming the arguments passed to p2p.Protocol.Run function
// the third argument is the CodeMap describing the protocol messages and options
func NewPeer(p *p2p.Peer, rw p2p.MsgReadWriter, ct *CodeMap, m Messenger, disconn func()) *Peer {
	return &Peer{
		ct:         ct,
		m:          m,
		Peer:       p,
		rw:         rw,
		handlers:   make(map[reflect.Type][]func(interface{}) error),
		disconnect: disconn,
	}
}

// Register is called on the peer typically within the constructor of service instances running on peer connections
// These constructors are called by the p2p.Protocol#Run function
// It ties handler callbackss for specific message types
// A message type can have several handlers registered by the same or different protocol services
// Register is meant to be called once, deregistering is not currently supported therefore
// handlers are assumed to be static across handshake renegotiations
// i.e., a service instance either handles a message or not (irrespective of the handshake)
// it panics if the message type is not defined in the CodeMap
func (self *Peer) Register(msg interface{}, handler func(interface{}) error) uint {
	typ := reflect.TypeOf(msg)
	code, found := self.ct.messages[typ]
	if !found {
		panic(fmt.Sprintf("message type '%v' unknown ", typ))
	}
	glog.V(logger.Debug).Infof("registered handle for %v %v", msg, typ)
	self.handlers[typ] = append(self.handlers[typ], handler)
	return code
}

// Run starts the forever loop that handles incoming messages
// called within the p2p.Protocol#Run function
func (self *Peer) Run() error {
	var err error
	for {
		_, err = self.handleIncoming()
		if err != nil {
			return err
		}
	}
}

// Drop disconnects a peer.
// falls back to self.disconnect which is set as p2p.Peer#Disconnect except
// for test peers where it calls p2p.MsgPipe#Close so that the readloop can terminate
// TODO: may need to implement protocol drop only? don't want to kick off the peer
// if they are useful for other protocols
// overwrite Disconnect for testing, so that protocol readloop quits
func (self *Peer) Drop() {
	self.disconnect()
}

// Send takes a message, encodes it in RLP, finds the right message code and sends the
// message off to the peer
// this low level call will be wrapped by libraries providing routed or broadcast sends
// but often just used to forward and push messages to directly connected peers
func (self *Peer) Send(msg interface{}) error {
	typ := reflect.TypeOf(msg)
	code, found := self.ct.messages[typ]
	if !found {
		return errorf(ErrInvalidMsgType, "%v", typ)
	}
	glog.V(logger.Debug).Infof("=> %v %v (%d)", msg, typ, code)
	err := self.m.SendMsg(self.rw, uint64(code), msg)
	if err != nil {
		self.Drop()
		return errorf(ErrWrite, "(msg code: %v): %v", code, err)
	}
	return nil
}

// handleIncoming(code)
// is called each cycle of the main forever loop that handles and dispatches incoming messages
// if this returns an error the loop returns and the peer is disconnected with the error
// checks message size, out-of-range message codes, handles decoding with reflection,
// call handlers as callback onside
func (self *Peer) handleIncoming() (interface{}, error) {
	msg, err := self.m.ReadMsg(self.rw)
	if err != nil {
		return nil, err
	}
	glog.V(logger.Debug).Infof("<= %v", msg)
	// make sure that the payload has been fully consumed
	defer msg.Discard()

	if msg.Size > uint32(self.ct.MaxMsgSize) {
		return nil, errorf(ErrMsgTooLong, "%v > %v", msg.Size, self.ct.MaxMsgSize)
	}

	// check if the message code is correct
	maxMsgCode := uint(len(self.ct.messages))
	if msg.Code >= uint64(maxMsgCode) {
		return nil, errorf(ErrInvalidMsgCode, "%v (>=%v)", msg.Code, maxMsgCode)
	}

	// it is safe to be unsafe here
	typ := self.ct.codes[msg.Code]
	val := reflect.New(typ)
	req := val.Elem()
	req.Set(reflect.Zero(typ))
	if err := msg.Decode(val.Interface()); err != nil {
		return nil, errorf(ErrDecode, "<= %v: %v", msg, err)
	}
	glog.V(logger.Debug).Infof("<= %v %v (%d)", req, typ, msg.Code)

	// call the registered handler callbacks
	// a registered callback take the decoded message as argument as an interface
	// which the handler is supposed to cast to the appropriate type
	// it is entirely safe not to check the cast in the handler since the handler is
	// chosen based on the proper type in the first place
	handlers := self.handlers[typ]
	if len(handlers) == 0 {
		glog.V(6).Infof("no handler (msg code %v)", msg.Code)
		// return nil, errorf(ErrNoHandler, "(msg code %v)", msg.Code)
	} else {
		for i, f := range handlers {
			glog.V(6).Infof("handler %v for %v", i, typ)
			err = f(req.Interface())
			if err != nil {
				return nil, errorf(ErrHandler, "(msg code %v): %v", msg.Code, err)
			}
		}
	}
	return req.Interface(), nil
}

// Handshake initiates a handshake on the peer connection
// * the argument is the local  handshake 	to be sent to the remote peer
// * expects a remote handshake back of the same type
// returns the remote hs and an error
func (self *Peer) Handshake(hs interface{}) (interface{}, error) {
	typ := reflect.TypeOf(hs)
	_, found := self.ct.messages[typ]
	if !found {
		return nil, errorf(ErrLocalHandshake, "unknown handshake message type: %v", typ)
	}
	errc := make(chan error)
	go func() {
		err := self.Send(hs)
		if err != nil {
			err = errorf(ErrLocalHandshake, "cannot send: %v", err)
		}
		errc <- err
	}()
	// receiving and validating remote handshake, expect code
	rhs, err := self.handleIncoming()
	if err != nil {
		return nil, errorf(ErrRemoteHandshake, "'%v': %v", self.ct.Name, err)
	}
	err = <-errc
	return rhs, err
}
