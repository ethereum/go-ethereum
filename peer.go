package main

import (
	"github.com/ethereum/ethutil-go"
	"github.com/ethereum/ethwire-go"
	"log"
	"net"
	"sync/atomic"
	"time"
)

const (
	// The size of the output buffer for writing messages
	outputBufferSize             = 50
)

type Peer struct {
	// Server interface
	server *Server
	// Net connection
	conn net.Conn
	// Output queue which is used to communicate and handle messages
	outputQueue chan *ethwire.InOutMsg
	// Quit channel
	quit chan bool
	// Determines whether it's an inbound or outbound peer
	inbound bool
	// Flag for checking the peer's connectivity state
	connected int32
	disconnect int32
	lastSend time.Time
	versionKnown bool
}

func NewPeer(conn net.Conn, server *Server, inbound bool) *Peer {
	return &Peer{
		outputQueue: make(chan *ethwire.InOutMsg, outputBufferSize),
		quit:        make(chan bool),
		server:      server,
		conn:        conn,
		inbound:     inbound,
		disconnect:  0,
		connected:   1,
	}
}

func NewOutboundPeer(addr string, server *Server) *Peer {
	p := &Peer{
		outputQueue: make(chan *ethwire.InOutMsg, outputBufferSize),
		quit:        make(chan bool),
		server:      server,
		inbound:     false,
		connected:   0,
		disconnect:  1,
	}

	// Set up the connection in another goroutine so we don't block the main thread
	go func() {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			p.Stop()
		}
		p.conn = conn

		// Atomically set the connection state
		atomic.StoreInt32(&p.connected, 1)
		atomic.StoreInt32(&p.disconnect, 0)

		log.Println("Connected to peer ::", conn.RemoteAddr())

		p.Start()
	}()

	return p
}

// Outputs any RLP encoded data to the peer
func (p *Peer) QueueMessage(msg *ethwire.InOutMsg) {
	p.outputQueue <- msg
}

func (p *Peer) writeMessage(msg *ethwire.InOutMsg) {
	// Ignore the write if we're not connected
	if atomic.LoadInt32(&p.connected) != 1 {
		return
	}

	if !p.versionKnown {
		switch msg.MsgType {
		case "verack": // Ok
		default: // Anything but ack is allowed
			return
		}
	}

	err := ethwire.WriteMessage(p.conn, msg)
	if err != nil {
		log.Println("Can't send message:", err)
		// Stop the client if there was an error writing to it
		p.Stop()
		return
	}
}

// Outbound message handler. Outbound messages are handled here
func (p *Peer) HandleOutbound() {
out:
	for {
		select {
		// Main message queue. All outbound messages are processed through here
		case msg := <-p.outputQueue:
			p.writeMessage(msg)

			p.lastSend = time.Now()
		// Break out of the for loop if a quit message is posted
		case <-p.quit:
			break out
		}
	}

clean:
	// This loop is for draining the output queue and anybody waiting for us
	for {
		select {
		case <- p.outputQueue:
			// TODO
		default:
			break clean
		}
	}
}

// Inbound handler. Inbound messages are received here and passed to the appropriate methods
func (p *Peer) HandleInbound() {

out:
	for atomic.LoadInt32(&p.disconnect) == 0 {
		// Wait for a message from the peer
		msg, err := ethwire.ReadMessage(p.conn)
		if err != nil {
			log.Println(err)

			break out
		}

		if Debug {
			log.Printf("Received %s\n", msg.MsgType)
		}

		// TODO Hash data and check if for existence (= ignore)

		switch msg.MsgType {
		case "verack":
			// Version message
			p.handleVersionAck(msg)
		case "block":
			err := p.server.blockManager.ProcessBlock(ethutil.NewBlock(msg.Data))
			if err != nil {
				log.Println(err)
			}
		}
	}

	p.Stop()
}

func (p *Peer) Start() {
	if !p.inbound {
		err := p.pushVersionAck()
		if err != nil {
			log.Printf("Peer can't send outbound version ack", err)

			p.Stop()
		}
	}

	// Run the outbound handler in a new goroutine
	go p.HandleOutbound()
	// Run the inbound handler in a new goroutine
	go p.HandleInbound()
}

func (p *Peer) Stop() {
	if atomic.AddInt32(&p.disconnect, 1) != 1 {
		return
	}

	close(p.quit)
	if atomic.LoadInt32(&p.connected) != 0 {
		p.conn.Close()
	}

	log.Println("Peer shutdown")
}

func (p *Peer) pushVersionAck() error {
	msg := ethwire.NewMessage("verack", p.server.Nonce, []byte("01"))

	p.QueueMessage(msg)

	return nil
}

func (p *Peer) handleVersionAck(msg *ethwire.InOutMsg) {
	// Detect self connect
	if msg.Nonce == p.server.Nonce {
		log.Println("Peer connected to self, disconnecting")

		p.Stop()

		return
	}

	p.versionKnown = true

	// If this is an inbound connection send an ack back
	if p.inbound {
		err := p.pushVersionAck()
		if err != nil {
			log.Println("Peer can't send ack back")

			p.Stop()
		}
	}
}
