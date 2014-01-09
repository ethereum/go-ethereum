package main

import (
  "net"
  "errors"
  "log"
)

type InMsg struct {
  msgType   string  // Specifies how the encoded data should be interpreted
  data      []byte  // RLP encoded data
}

func ReadMessage(conn net.Conn) (*InMsg, error) {
  buff := make([]byte, 4069)

  // Wait for a message from this peer
  n, err := conn.Read(buff)
  if err != nil {
    return nil, err
  } else if n == 0 {
    return nil, errors.New("Empty message received")
  }

  // Read the header (MAX n)
  decoder := NewRlpDecoder(buff[:n])
  t := decoder.Get(0).AsString()
  if t == "" {
    return nil, errors.New("Data contained no data type")
  }

  return &InMsg{msgType: t, data: decoder.Get(1).AsBytes()}, nil
}

type OutMsg struct {
  data      []byte
}

type Peer struct {
  server      *Server
  conn        net.Conn
  outputQueue chan OutMsg
  quit        chan bool
}

func NewPeer(conn net.Conn, server *Server) *Peer {
  return &Peer{
    outputQueue:       make(chan OutMsg, 1),  // Buffered chan of 1 is enough
    quit:              make(chan bool),

    server:            server,
    conn:              conn,
  }
}

// Outputs any RLP encoded data to the peer
func (p *Peer) QueueMessage(data []byte) {
  p.outputQueue <- OutMsg{data: data}
}

func (p *Peer) HandleOutbound() {
out:
  for {
    switch {
    case <- p.quit:
      break out
    }
  }
}

func (p *Peer) HandleInbound() {
  defer p.conn.Close()

out:
  for {
    msg, err := ReadMessage(p.conn)
    if err != nil {
      log.Println(err)

      break out
    }

    log.Println(msg)
  }

  // Notify the out handler we're quiting
  p.quit <- true
}

func (p *Peer) Start() {
  go p.HandleOutbound()
  go p.HandleInbound()
}
