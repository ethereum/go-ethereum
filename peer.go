package main

import (
	"github.com/ethereum/ethutil-go"
	"github.com/ethereum/ethwire-go"
	"log"
	"net"
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

	inbound bool // Determines whether it's an inbound or outbound peer
}

func NewPeer(conn net.Conn, server *Server, inbound bool) *Peer {
	return &Peer{
		outputQueue: make(chan *ethwire.InOutMsg, 1), // Buffered chan of 1 is enough
		quit:        make(chan bool),
		server:      server,
		conn:        conn,
		inbound:     inbound,
	}
}

// Outputs any RLP encoded data to the peer
func (p *Peer) QueueMessage(msg *ethwire.InOutMsg) {
	p.outputQueue <- msg //ethwire.InOutMsg{MsgType: msgType, Nonce: ethutil.RandomUint64(), Data: data}
}

// Outbound message handler. Outbound messages are handled here
func (p *Peer) HandleOutbound() {
out:
	for {
		select {
		// Main message queue. All outbound messages are processed through here
		case msg := <-p.outputQueue:
			// TODO Message checking and handle accordingly
			err := ethwire.WriteMessage(p.conn, msg)
			if err != nil {
				log.Println(err)

				// Stop the client if there was an error writing to it
				p.Stop()
			}

		// Break out of the for loop if a quit message is posted
		case <-p.quit:
			break out
		}
	}
}

// Inbound handler. Inbound messages are received here and passed to the appropriate methods
func (p *Peer) HandleInbound() {
	defer p.Stop()

out:
	for {
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

	// Notify the out handler we're quiting
	p.quit <- true
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
	p.conn.Close()

	p.quit <- true
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

	log.Println("mnonce", msg.Nonce, "snonce", p.server.Nonce)

	// If this is an inbound connection send an ack back
	if p.inbound {
		err := p.pushVersionAck()
		if err != nil {
			log.Println("Peer can't send ack back")

			p.Stop()
		}
	}
}
