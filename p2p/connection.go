package p2p

import (
	"bytes"
	// "fmt"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/ethutil"
)

type Connection struct {
	conn net.Conn
	// conn       NetworkConnection
	timeout    time.Duration
	in         chan []byte
	out        chan []byte
	err        chan *PeerError
	closingIn  chan chan bool
	closingOut chan chan bool
}

// const readBufferLength = 2 //for testing

const readBufferLength = 1440
const partialsQueueSize = 10
const maxPendingQueueSize = 1
const defaultTimeout = 500

var magicToken = []byte{34, 64, 8, 145}

func (self *Connection) Open() {
	go self.startRead()
	go self.startWrite()
}

func (self *Connection) Close() {
	self.closeIn()
	self.closeOut()
}

func (self *Connection) closeIn() {
	errc := make(chan bool)
	self.closingIn <- errc
	<-errc
}

func (self *Connection) closeOut() {
	errc := make(chan bool)
	self.closingOut <- errc
	<-errc
}

func NewConnection(conn net.Conn, errchan chan *PeerError) *Connection {
	return &Connection{
		conn:       conn,
		timeout:    defaultTimeout,
		in:         make(chan []byte),
		out:        make(chan []byte),
		err:        errchan,
		closingIn:  make(chan chan bool, 1),
		closingOut: make(chan chan bool, 1),
	}
}

func (self *Connection) Read() <-chan []byte {
	return self.in
}

func (self *Connection) Write() chan<- []byte {
	return self.out
}

func (self *Connection) Error() <-chan *PeerError {
	return self.err
}

func (self *Connection) startRead() {
	payloads := make(chan []byte)
	done := make(chan *PeerError)
	pending := [][]byte{}
	var head []byte
	var wait time.Duration // initally 0 (no delay)
	read := time.After(wait * time.Millisecond)

	for {
		// if pending empty, nil channel blocks
		var in chan []byte
		if len(pending) > 0 {
			in = self.in // enable send case
			head = pending[0]
		} else {
			in = nil
		}

		select {
		case <-read:
			go self.read(payloads, done)
		case err := <-done:
			if err == nil { // no error but nothing to read
				if len(pending) < maxPendingQueueSize {
					wait = 100
				} else if wait == 0 {
					wait = 100
				} else {
					wait = 2 * wait
				}
			} else {
				self.err <- err // report error
				wait = 100
			}
			read = time.After(wait * time.Millisecond)
		case payload := <-payloads:
			pending = append(pending, payload)
			if len(pending) < maxPendingQueueSize {
				wait = 0
			} else {
				wait = 100
			}
			read = time.After(wait * time.Millisecond)
		case in <- head:
			pending = pending[1:]
		case errc := <-self.closingIn:
			errc <- true
			close(self.in)
			return
		}

	}
}

func (self *Connection) startWrite() {
	pending := [][]byte{}
	done := make(chan *PeerError)
	writing := false
	for {
		if len(pending) > 0 && !writing {
			writing = true
			go self.write(pending[0], done)
		}
		select {
		case payload := <-self.out:
			pending = append(pending, payload)
		case err := <-done:
			if err == nil {
				pending = pending[1:]
				writing = false
			} else {
				self.err <- err // report error
			}
		case errc := <-self.closingOut:
			errc <- true
			close(self.out)
			return
		}
	}
}

func pack(payload []byte) (packet []byte) {
	length := ethutil.NumberToBytes(uint32(len(payload)), 32)
	// return error if too long?
	// Write magic token and payload length (first 8 bytes)
	packet = append(magicToken, length...)
	packet = append(packet, payload...)
	return
}

func avoidPanic(done chan *PeerError) {
	if rec := recover(); rec != nil {
		err := NewPeerError(MiscError, " %v", rec)
		logger.Debugln(err)
		done <- err
	}
}

func (self *Connection) write(payload []byte, done chan *PeerError) {
	defer avoidPanic(done)
	var err *PeerError
	_, ok := self.conn.Write(pack(payload))
	if ok != nil {
		err = NewPeerError(WriteError, " %v", ok)
		logger.Debugln(err)
	}
	done <- err
}

func (self *Connection) read(payloads chan []byte, done chan *PeerError) {
	//defer avoidPanic(done)

	partials := make(chan []byte, partialsQueueSize)
	errc := make(chan *PeerError)
	go self.readPartials(partials, errc)

	packet := []byte{}
	length := 8
	start := true
	var err *PeerError
out:
	for {
		// appends partials read via connection until packet is
		// - either parseable (>=8bytes)
		// - or complete (payload fully consumed)
		for len(packet) < length {
			partial, ok := <-partials
			if !ok { // partials channel is closed
				err = <-errc
				if err == nil && len(packet) > 0 {
					if start {
						err = NewPeerError(PacketTooShort, "%v", packet)
					} else {
						err = NewPeerError(PayloadTooShort, "%d < %d", len(packet), length)
					}
				}
				break out
			}
			packet = append(packet, partial...)
		}
		if start {
			// at least 8 bytes read, can validate packet
			if bytes.Compare(magicToken, packet[:4]) != 0 {
				err = NewPeerError(MagicTokenMismatch, " received %v", packet[:4])
				break
			}
			length = int(ethutil.BytesToNumber(packet[4:8]))
			packet = packet[8:]

			if length > 0 {
				start = false // now consuming payload
			} else { //penalize peer but read on
				self.err <- NewPeerError(EmptyPayload, "")
				length = 8
			}
		} else {
			// packet complete (payload fully consumed)
			payloads <- packet[:length]
			packet = packet[length:] // resclice packet
			start = true
			length = 8
		}
	}

	// this stops partials read via the connection, should we?
	//if err != nil {
	//  select {
	//    case errc <- err
	//  default:
	//}
	done <- err
}

func (self *Connection) readPartials(partials chan []byte, errc chan *PeerError) {
	defer close(partials)
	for {
		// Give buffering some time
		self.conn.SetReadDeadline(time.Now().Add(self.timeout * time.Millisecond))
		buffer := make([]byte, readBufferLength)
		// read partial from connection
		bytesRead, err := self.conn.Read(buffer)
		if err == nil || err.Error() == "EOF" {
			if bytesRead > 0 {
				partials <- buffer[:bytesRead]
			}
			if err != nil && err.Error() == "EOF" {
				break
			}
		} else {
			// unexpected error, report to errc
			err := NewPeerError(ReadError, " %v", err)
			logger.Debugln(err)
			errc <- err
			return // will close partials channel
		}
	}
	close(errc)
}
