package main

import (
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
	outputQueue chan ethwire.InOutMsg
	// Quit channel
	quit chan bool
}

func NewPeer(conn net.Conn, server *Server) *Peer {
	return &Peer{
		outputQueue: make(chan ethwire.InOutMsg, 1), // Buffered chan of 1 is enough
		quit:        make(chan bool),

		server: server,
		conn:   conn,
	}
}

// Outputs any RLP encoded data to the peer
func (p *Peer) QueueMessage(msgType string, data []byte) {
	p.outputQueue <- ethwire.InOutMsg{MsgType: msgType, Data: data}
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

		// TODO
		data, _ := Decode(msg.Data, 0)
		log.Printf("%s, %s\n", msg.MsgType, data)
	}

	// Notify the out handler we're quiting
	p.quit <- true
}

func (p *Peer) Start() {
	// Run the outbound handler in a new goroutine
	go p.HandleOutbound()
	// Run the inbound handler in a new goroutine
	go p.HandleInbound()
}

func (p *Peer) Stop() {
	p.conn.Close()

	p.quit <- true
}
