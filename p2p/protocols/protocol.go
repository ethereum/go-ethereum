// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

/*
Package protocols is an extension to p2p. It offers a user friendly simple way to define
devp2p subprotocols by abstracting away code standardly shared by protocols.

* automate assigments of code indexes to messages
* automate RLP decoding/encoding based on reflecting
* provide the forever loop to read incoming messages
* standardise error handling related to communication
* standardised	handshake negotiation
* TODO: automatic generation of wire protocol specification for peers

*/
package protocols

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"reflect"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/swarm/spancontext"
	"github.com/ethereum/go-ethereum/swarm/tracing"
	opentracing "github.com/opentracing/opentracing-go"
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

func (e Error) Error() (message string) {
	if len(e.message) == 0 {
		name, ok := errorToString[e.Code]
		if !ok {
			panic("invalid message code")
		}
		e.message = name
		if e.format != "" {
			e.message += ": " + fmt.Sprintf(e.format, e.params...)
		}
	}
	return e.message
}

func errorf(code int, format string, params ...interface{}) *Error {
	return &Error{
		Code:   code,
		format: format,
		params: params,
	}
}

// WrappedMsg is used to propagate marshalled context alongside message payloads
type WrappedMsg struct {
	Context []byte
	Size    uint32
	Payload []byte
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

	// Messages is a list of message data types which this protocol uses, with
	// each message type being sent with its array index as the code (so
	// [&foo{}, &bar{}, &baz{}] would send foo, bar and baz with codes
	// 0, 1 and 2 respectively)
	// each message must have a single unique data type
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

// Length returns the number of message types in the protocol
func (s *Spec) Length() uint64 {
	return uint64(len(s.Messages))
}

// GetCode returns the message code of a type, and boolean second argument is
// false if the message type is not found
func (s *Spec) GetCode(msg interface{}) (uint64, bool) {
	s.init()
	typ := reflect.TypeOf(msg)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	code, ok := s.codes[typ]
	return code, ok
}

// NewMsg construct a new message type given the code
func (s *Spec) NewMsg(code uint64) (interface{}, bool) {
	s.init()
	typ, ok := s.types[code]
	if !ok {
		return nil, false
	}
	return reflect.New(typ).Interface(), true
}

// Peer represents a remote peer or protocol instance that is running on a peer connection with
// a remote peer
type Peer struct {
	*p2p.Peer                   // the p2p.Peer object representing the remote
	rw        p2p.MsgReadWriter // p2p.MsgReadWriter to send messages to and read messages from
	spec      *Spec
}

// NewPeer constructs a new peer
// this constructor is called by the p2p.Protocol#Run function
// the first two arguments are the arguments passed to p2p.Protocol.Run function
// the third argument is the Spec describing the protocol
func NewPeer(p *p2p.Peer, rw p2p.MsgReadWriter, spec *Spec) *Peer {
	return &Peer{
		Peer: p,
		rw:   rw,
		spec: spec,
	}
}

// Run starts the forever loop that handles incoming messages
// called within the p2p.Protocol#Run function
// the handler argument is a function which is called for each message received
// from the remote peer, a returned error causes the loop to exit
// resulting in disconnection
func (p *Peer) Run(handler func(ctx context.Context, msg interface{}) error) error {
	for {
		if err := p.handleIncoming(handler); err != nil {
			if err != io.EOF {
				metrics.GetOrRegisterCounter("peer.handleincoming.error", nil).Inc(1)
				log.Error("peer.handleIncoming", "err", err)
			}

			return err
		}
	}
}

// Drop disconnects a peer.
// TODO: may need to implement protocol drop only? don't want to kick off the peer
// if they are useful for other protocols
func (p *Peer) Drop(err error) {
	p.Disconnect(p2p.DiscSubprotocolError)
}

// Send takes a message, encodes it in RLP, finds the right message code and sends the
// message off to the peer
// this low level call will be wrapped by libraries providing routed or broadcast sends
// but often just used to forward and push messages to directly connected peers
func (p *Peer) Send(ctx context.Context, msg interface{}) error {
	defer metrics.GetOrRegisterResettingTimer("peer.send_t", nil).UpdateSince(time.Now())
	metrics.GetOrRegisterCounter("peer.send", nil).Inc(1)

	var b bytes.Buffer
	if tracing.Enabled {
		writer := bufio.NewWriter(&b)

		tracer := opentracing.GlobalTracer()

		sctx := spancontext.FromContext(ctx)

		if sctx != nil {
			err := tracer.Inject(
				sctx,
				opentracing.Binary,
				writer)
			if err != nil {
				return err
			}
		}

		writer.Flush()
	}

	r, err := rlp.EncodeToBytes(msg)
	if err != nil {
		return err
	}

	wmsg := WrappedMsg{
		Context: b.Bytes(),
		Size:    uint32(len(r)),
		Payload: r,
	}

	code, found := p.spec.GetCode(msg)
	if !found {
		return errorf(ErrInvalidMsgType, "%v", code)
	}
	return p2p.Send(p.rw, code, wmsg)
}

// handleIncoming(code)
// is called each cycle of the main forever loop that dispatches incoming messages
// if this returns an error the loop returns and the peer is disconnected with the error
// this generic handler
// * checks message size,
// * checks for out-of-range message codes,
// * handles decoding with reflection,
// * call handlers as callbacks
func (p *Peer) handleIncoming(handle func(ctx context.Context, msg interface{}) error) error {
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	// make sure that the payload has been fully consumed
	defer msg.Discard()

	if msg.Size > p.spec.MaxMsgSize {
		return errorf(ErrMsgTooLong, "%v > %v", msg.Size, p.spec.MaxMsgSize)
	}

	// unmarshal wrapped msg, which might contain context
	var wmsg WrappedMsg
	err = msg.Decode(&wmsg)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	ctx := context.Background()

	// if tracing is enabled and the context coming within the request is
	// not empty, try to unmarshal it
	if tracing.Enabled && len(wmsg.Context) > 0 {
		var sctx opentracing.SpanContext

		tracer := opentracing.GlobalTracer()
		sctx, err = tracer.Extract(
			opentracing.Binary,
			bytes.NewReader(wmsg.Context))
		if err != nil {
			log.Error(err.Error())
			return err
		}

		ctx = spancontext.WithContext(ctx, sctx)
	}

	val, ok := p.spec.NewMsg(msg.Code)
	if !ok {
		return errorf(ErrInvalidMsgCode, "%v", msg.Code)
	}
	if err := rlp.DecodeBytes(wmsg.Payload, val); err != nil {
		return errorf(ErrDecode, "<= %v: %v", msg, err)
	}

	// call the registered handler callbacks
	// a registered callback take the decoded message as argument as an interface
	// which the handler is supposed to cast to the appropriate type
	// it is entirely safe not to check the cast in the handler since the handler is
	// chosen based on the proper type in the first place
	if err := handle(ctx, val); err != nil {
		return errorf(ErrHandler, "(msg code %v): %v", msg.Code, err)
	}
	return nil
}

// Handshake negotiates a handshake on the peer connection
// * arguments
//   * context
//   * the local handshake to be sent to the remote peer
//   * funcion to be called on the remote handshake (can be nil)
// * expects a remote handshake back of the same type
// * the dialing peer needs to send the handshake first and then waits for remote
// * the listening peer waits for the remote handshake and then sends it
// returns the remote handshake and an error
func (p *Peer) Handshake(ctx context.Context, hs interface{}, verify func(interface{}) error) (rhs interface{}, err error) {
	if _, ok := p.spec.GetCode(hs); !ok {
		return nil, errorf(ErrHandshake, "unknown handshake message type: %T", hs)
	}
	errc := make(chan error, 2)
	handle := func(ctx context.Context, msg interface{}) error {
		rhs = msg
		if verify != nil {
			return verify(rhs)
		}
		return nil
	}
	send := func() { errc <- p.Send(ctx, hs) }
	receive := func() { errc <- p.handleIncoming(handle) }

	go func() {
		if p.Inbound() {
			receive()
			send()
		} else {
			send()
			receive()
		}
	}()

	for i := 0; i < 2; i++ {
		select {
		case err = <-errc:
		case <-ctx.Done():
			err = ctx.Err()
		}
		if err != nil {
			return nil, errorf(ErrHandshake, err.Error())
		}
	}
	return rhs, nil
}
