package main

import (
  "container/list"
  "net"
  "log"
  _"time"
)

var Db *LDBDatabase

type Server struct {
  // Channel for shutting down the server
  shutdownChan chan bool
  // DB interface
  db          *LDBDatabase
  // Block manager for processing new blocks and managing the block chain
  blockManager *BlockManager
  // Peers (NYI)
  peers       *list.List
}

func NewServer() (*Server, error) {
  db, err := NewLDBDatabase()
  if err != nil {
    return nil, err
  }

  Db = db

  server := &Server{
    shutdownChan:    make(chan bool),
    blockManager:    NewBlockManager(),
    db:              db,
    peers:           list.New(),
  }

  return server, nil
}

func (s *Server) AddPeer(conn net.Conn) {
  peer := NewPeer(conn, s)
  s.peers.PushBack(peer)
  peer.Start()

  log.Println("Peer connected ::", conn.RemoteAddr())
}

func (s *Server) ConnectToPeer(addr string) error {
  conn, err := net.Dial("tcp", addr)

  if err != nil {
    return err
  }

  peer := NewPeer(conn, s)
  s.peers.PushBack(peer)
  peer.Start()


  log.Println("Connected to peer ::", conn.RemoteAddr())

  return nil
}

func (s *Server) Broadcast(msgType string, data []byte) {
  for e := s.peers.Front(); e != nil; e = e.Next() {
    if peer, ok := e.Value.(*Peer); ok {
      peer.QueueMessage(msgType, data)
    }
  }
}

// Start the server
func (s *Server) Start() {
  // For now this function just blocks the main thread
  ln, err := net.Listen("tcp", ":12345")
  if err != nil {
    log.Fatal(err)
  }

  go func() {
    for {
      conn, err := ln.Accept()
      if err != nil {
        log.Println(err)
        continue
      }

      go s.AddPeer(conn)
    }
  }()

  // TMP
  //go func() {
  //  for {
  //    s.Broadcast("block", Encode("blockdata"))
//
//      time.Sleep(100 * time.Millisecond)
//    }
//  }()
}

func (s *Server) Stop() {
  // Close the database
  defer s.db.Close()

  // Loop thru the peers and close them (if we had them)
  for e := s.peers.Front(); e != nil; e = e.Next() {
    if peer, ok := e.Value.(*Peer); ok {
      peer.Stop()
    }
  }

  s.shutdownChan <- true
}

// This function will wait for a shutdown and resumes main thread execution
func (s *Server) WaitForShutdown() {
  <- s.shutdownChan
}
