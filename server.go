package main

import (
  "container/list"
  "net"
  "log"
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
  s.peers.PushBack(NewPeer(conn, s))
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
}

func (s *Server) Stop() {
  // Close the database
  defer s.db.Close()

  // Loop thru the peers and close them (if we had them)
  for e := s.peers.Front(); e != nil; e = e.Next() {
    // peer close etc
  }

  s.shutdownChan <- true
}

// This function will wait for a shutdown and resumes main thread execution
func (s *Server) WaitForShutdown() {
  <- s.shutdownChan
}
