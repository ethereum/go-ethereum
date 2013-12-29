package main

import (
  "container/list"
  "time"
)

type Server struct {
  // Channel for shutting down the server
  shutdownChan chan bool
  // DB interface
  db          *Database
  // Peers (NYI)
  peers       *list.List
}

func NewServer() (*Server, error) {
  db, err := NewDatabase()
  if err != nil {
    return nil, err
  }

  server := &Server{
    shutdownChan:    make(chan bool),
    db:              db,
    peers:           list.New(),
  }

  return server, nil
}

// Start the server
func (s *Server) Start() {
  // For now this function just blocks the main thread
  for {
    time.Sleep( time.Second )
  }
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
