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
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
)

// error codes used by this  protocol scheme
const (
	ErrMsgTooLong = iota
	ErrDecode
	ErrWrite
	ErrInvalidMsgCode
	ErrInvalidMsgType
	ErrHandshake
	ErrNoHandler
	ErrHandler
)

// error description strings associated with the codes
var errorToString = map[int]string{
	ErrMsgTooLong:     "Message too long",
	ErrDecode:         "Invalid message (RLP error)",
	ErrWrite:          "Error sending message",
	ErrInvalidMsgCode: "Invalid message code",
	ErrInvalidMsgType: "Invalid message type",
	ErrHandshake:      "Handshake error",
	ErrNoHandler:      "No handler registered error",
	ErrHandler:        "Message handler error",
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

// Spec is a protocol specification including its name and version as well as
// the types of messages which are exchanged
type Spec struct {
	// Name is the name of the protocol, often a three-letter word
	Name string

	// Version is the version number of the protocol
	Version uint

	// MaxMsgSize is the maximum accepted length of the message payload
	MaxMsgSize uint32

	// Messages is a list of message types which this protocol uses, with
	// each message type being sent with its array index as the code (so
	// [&foo{}, &bar{}, &baz{}] would send foo, bar and baz with codes
	// 0, 1 and 2 respectively)
	Messages []interface{}

	initOnce sync.Once
	codes    map[reflect.Type]uint64
	types    map[uint64]reflect.Type
}

func (s *Spec) init() {
	s.initOnce.Do(func() {
		s.codes = make(map[reflect.Type]uint64, len(s.Messages))
		s.types = make(map[uint64]reflect.Type, len(s.Messages))
		for i, msg := range s.Messages {
			code := uint64(i)
			typ := reflect.TypeOf(msg)
			if typ.Kind() == reflect.Ptr {
				typ = typ.Elem()
			}
			s.codes[typ] = code
			s.types[code] = typ
		}
	})
}

func (s *Spec) Length() uint64 {
	return uint64(len(s.Messages))
}

func (s *Spec) GetCode(msg interface{}) (uint64, bool) {
	s.init()
	typ := reflect.TypeOf(msg)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	code, ok := s.codes[typ]
	return code, ok
}

func (s *Spec) NewMsg(code uint64) (interface{}, bool) {
	s.init()
	typ, ok := s.types[code]
	if !ok {
		return nil, false
	}
	return reflect.New(typ).Interface(), true
}

// A Peer represents a remote peer or protocol instance that is running on a peer connection with
// a remote peer
type Peer struct {
	*p2p.Peer                   // the p2p.Peer object representing the remote
	rw        p2p.MsgReadWriter // p2p.MsgReadWriter to send messages to and read messages from
	spec      *Spec
	Errc      chan error
	wErrc     chan error // write error channel
}

// NewPeer returns a new peer
// this constructor is called by the p2p.Protocol#Run function
// the first two arguments are comming the arguments passed to p2p.Protocol.Run function
// the third argument is the CodeMap describing the protocol messages and options
func NewPeer(p *p2p.Peer, rw p2p.MsgReadWriter, spec *Spec) *Peer {
	return &Peer{
		Peer:  p,
		rw:    rw,
		spec:  spec,
		Errc:  make(chan error),
		wErrc: make(chan error),
	}
}

// Run starts the forever loop that handles incoming messages
// called within the p2p.Protocol#Run function
func (self *Peer) Run(handler func(msg interface{}) error) error {
	go func() {
		for {
			if err := self.handleIncoming(handler); err != nil {
				self.Errc <- err
				return
			}
		}
	}()
	return <-self.Errc
}

// Drop disconnects a peer.
// falls back to self.disconnect which is set as p2p.Peer#Disconnect except
// for test peers where it calls p2p.MsgPipe#Close so that the readloop can terminate
// TODO: may need to implement protocol drop only? don't want to kick off the peer
// if they are useful for other protocols
// overwrite Disconnect for testing, so that protocol readloop quits
func (self *Peer) Drop(err error) {
	self.Errc <- err
}

// Send takes a message, encodes it in RLP, finds the right message code and sends the
// message off to the peer
// this low level call will be wrapped by libraries providing routed or broadcast sends
// but often just used to forward and push messages to directly connected peers
func (self *Peer) Send(msg interface{}) error {
	code, found := self.spec.GetCode(msg)
	if !found {
		return errorf(ErrInvalidMsgType, "%v", code)
	}
	log.Trace(fmt.Sprintf("=> msg #%d TO %v : %v", code, self.ID(), msg))
	return p2p.Send(self.rw, code, msg)
}

// handleIncoming(code)
// is called each cycle of the main forever loop that handles and dispatches incoming messages
// if this returns an error the loop returns and the peer is disconnected with the error
// checks message size, out-of-range message codes, handles decoding with reflection,
// call handlers as callback onside
func (self *Peer) handleIncoming(handle func(msg interface{}) error) error {
	msg, err := self.rw.ReadMsg()
	if err != nil {
		return err
	}
	log.Trace(fmt.Sprintf("<= %v", msg))
	// make sure that the payload has been fully consumed
	defer msg.Discard()

	if msg.Size > self.spec.MaxMsgSize {
		return errorf(ErrMsgTooLong, "%v > %v", msg.Size, self.spec.MaxMsgSize)
	}

	val, ok := self.spec.NewMsg(msg.Code)
	if !ok {
		return errorf(ErrInvalidMsgCode, "%v", msg.Code)
	}
	if err := msg.Decode(val); err != nil {
		return errorf(ErrDecode, "<= %v: %v", msg, err)
	}
	log.Trace(fmt.Sprintf("<= %v FROM %v %T %v", msg, self.ID(), val, val))

	// call the registered handler callbacks
	// a registered callback take the decoded message as argument as an interface
	// which the handler is supposed to cast to the appropriate type
	// it is entirely safe not to check the cast in the handler since the handler is
	// chosen based on the proper type in the first place
	if err := handle(val); err != nil {
		return errorf(ErrHandler, "(msg code %v): %v", msg.Code, err)
	}
	return nil
}

// Handshake initiates a handshake on the peer connection
// * the argument is the local  handshake 	to be sent to the remote peer
// * expects a remote handshake back of the same type
// returns the remote hs and an error
func (self *Peer) Handshake(ctx context.Context, hs interface{}) (interface{}, error) {
	if _, ok := self.spec.GetCode(hs); !ok {
		return nil, errorf(ErrHandshake, "unknown handshake message type: %T", hs)
	}
	errc := make(chan error, 2)
	go func() {
		if err := self.Send(hs); err != nil {
			errc <- errorf(ErrHandshake, "cannot send: %v", err)
		}
	}()
	hsc := make(chan interface{})
	go func() {
		var rhs interface{}
		err := self.handleIncoming(func(msg interface{}) error {
			rhs = msg
			return nil
		})
		if err != nil {
			errc <- err
			return
		}
		hsc <- rhs
	}()
	select {
	case rhs := <-hsc:
		return rhs, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errc:
		return nil, err
	}
}
