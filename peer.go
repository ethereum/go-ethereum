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

type OutMsg struct {
  msgType   string
  data      []byte
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
  // XXX The data specification is made up. This will change once more details have been released on the specification of the format
  decoder := NewRlpDecoder(buff[:n])
  t := decoder.Get(0).AsString()
  if t == "" {
    return nil, errors.New("Data contained no data type")
  }

  return &InMsg{msgType: t, data: decoder.Get(1).AsBytes()}, nil
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
func (p *Peer) QueueMessage(msgType string, data []byte) {
  p.outputQueue <- OutMsg{msgType: msgType, data: data}
}

func (p *Peer) HandleOutbound() {
out:
  for {
    select {
    case msg := <-p.outputQueue:
      p.WriteMessage(msg)

    case <- p.quit:
      break out
    }
  }
}

func (p *Peer) WriteMessage(msg OutMsg) {
  encoded := Encode([]interface{}{ msg.msgType, msg.data })
  _, err := p.conn.Write(encoded)
  if err != nil {
    log.Println(err)
    p.Stop()
  }
}

func (p *Peer) HandleInbound() {
  defer p.Stop()

out:
  for {
    msg, err := ReadMessage(p.conn)
    if err != nil {
      log.Println(err)

      break out
    }

    // TODO
    data, _ := Decode(msg.data, 0)
    log.Printf("%s, %s\n", msg.msgType, data)
  }

  // Notify the out handler we're quiting
  p.quit <- true
}

func (p *Peer) Start() {
  go p.HandleOutbound()
  go p.HandleInbound()
}

func (p *Peer) Stop() {
  defer p.conn.Close()

  p.quit <- true
}
