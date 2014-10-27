package p2p

import (
	"fmt"
	"sync"
	"time"
)

const (
	handlerTimeout = 1000
)

type Handlers map[string](func(p *Peer) Protocol)

type Messenger struct {
	conn          *Connection
	peer          *Peer
	handlers      Handlers
	protocolLock  sync.RWMutex
	protocols     []Protocol
	offsets       []MsgCode // offsets for adaptive message idss
	protocolTable map[string]int
	quit          chan chan bool
	err           chan *PeerError
	pulse         chan bool
}

func NewMessenger(peer *Peer, conn *Connection, errchan chan *PeerError, handlers Handlers) *Messenger {
	baseProtocol := NewBaseProtocol(peer)
	return &Messenger{
		conn:          conn,
		peer:          peer,
		offsets:       []MsgCode{baseProtocol.Offset()},
		handlers:      handlers,
		protocols:     []Protocol{baseProtocol},
		protocolTable: make(map[string]int),
		err:           errchan,
		pulse:         make(chan bool, 1),
		quit:          make(chan chan bool, 1),
	}
}

func (self *Messenger) Start() {
	self.conn.Open()
	go self.messenger()
	self.protocolLock.RLock()
	defer self.protocolLock.RUnlock()
	self.protocols[0].Start()
}

func (self *Messenger) Stop() {
	// close pulse to stop ping pong monitoring
	close(self.pulse)
	self.protocolLock.RLock()
	defer self.protocolLock.RUnlock()
	for _, protocol := range self.protocols {
		protocol.Stop() // could be parallel
	}
	q := make(chan bool)
	self.quit <- q
	<-q
	self.conn.Close()
}

func (self *Messenger) messenger() {
	in := self.conn.Read()
	for {
		select {
		case payload, ok := <-in:
			//dispatches message to the protocol asynchronously
			if ok {
				go self.handle(payload)
			} else {
				return
			}
		case q := <-self.quit:
			q <- true
			return
		}
	}
}

// handles each message by dispatching to the appropriate protocol
// using adaptive message codes
// this function is started as a separate go routine for each message
// it waits for the protocol response
// then encodes and sends outgoing messages to the connection's write channel
func (self *Messenger) handle(payload []byte) {
	// send ping to heartbeat channel signalling time of last message
	// select {
	// case self.pulse <- true:
	// default:
	// }
	self.pulse <- true
	// initialise message from payload
	msg, err := NewMsgFromBytes(payload)
	if err != nil {
		self.err <- NewPeerError(MiscError, " %v", err)
		return
	}
	// retrieves protocol based on message Code
	protocol, offset, peerErr := self.getProtocol(msg.Code())
	if err != nil {
		self.err <- peerErr
		return
	}
	// reset message code based on adaptive offset
	msg.Decode(offset)
	// dispatches
	response := make(chan *Msg)
	go protocol.HandleIn(msg, response)
	// protocol reponse timeout to prevent leaks
	timer := time.After(handlerTimeout * time.Millisecond)
	for {
		select {
		case outgoing, ok := <-response:
			// we check if response channel is not closed
			if ok {
				self.conn.Write() <- outgoing.Encode(offset)
			} else {
				return
			}
		case <-timer:
			return
		}
	}
}

// negotiated protocols
// stores offsets needed for adaptive message id scheme

// based on offsets set at handshake
// get the right protocol to handle the message
func (self *Messenger) getProtocol(code MsgCode) (Protocol, MsgCode, *PeerError) {
	self.protocolLock.RLock()
	defer self.protocolLock.RUnlock()
	base := MsgCode(0)
	for index, offset := range self.offsets {
		if code < offset {
			return self.protocols[index], base, nil
		}
		base = offset
	}
	return nil, MsgCode(0), NewPeerError(InvalidMsgCode, " %v", code)
}

func (self *Messenger) PingPong(timeout time.Duration, gracePeriod time.Duration, pingCallback func(), timeoutCallback func()) {
	fmt.Printf("pingpong keepalive started at %v", time.Now())

	timer := time.After(timeout)
	pinged := false
	for {
		select {
		case _, ok := <-self.pulse:
			if ok {
				pinged = false
				timer = time.After(timeout)
			} else {
				// pulse is closed, stop monitoring
				return
			}
		case <-timer:
			if pinged {
				fmt.Printf("timeout at %v", time.Now())
				timeoutCallback()
				return
			} else {
				fmt.Printf("pinged at %v", time.Now())
				pingCallback()
				timer = time.After(gracePeriod)
				pinged = true
			}
		}
	}
}

func (self *Messenger) AddProtocols(protocols []string) {
	self.protocolLock.Lock()
	defer self.protocolLock.Unlock()
	i := len(self.offsets)
	offset := self.offsets[i-1]
	for _, name := range protocols {
		protocolFunc, ok := self.handlers[name]
		if ok {
			protocol := protocolFunc(self.peer)
			self.protocolTable[name] = i
			i++
			offset += protocol.Offset()
			fmt.Println("offset ", name, offset)

			self.offsets = append(self.offsets, offset)
			self.protocols = append(self.protocols, protocol)
			protocol.Start()
		} else {
			fmt.Println("no ", name)
			// protocol not handled
		}
	}
}

func (self *Messenger) Write(protocol string, msg *Msg) error {
	self.protocolLock.RLock()
	defer self.protocolLock.RUnlock()
	i := 0
	offset := MsgCode(0)
	if len(protocol) > 0 {
		var ok bool
		i, ok = self.protocolTable[protocol]
		if !ok {
			return fmt.Errorf("protocol %v not handled by peer", protocol)
		}
		offset = self.offsets[i-1]
	}
	handler := self.protocols[i]
	// checking if protocol status/caps allows the message to be sent out
	if handler.HandleOut(msg) {
		self.conn.Write() <- msg.Encode(offset)
	}
	return nil
}
