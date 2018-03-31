// Package peerstream is a peer-to-peer networking library that multiplexes
// connections to many hosts. It attempts to simplify the complexity of:
//
// * accepting incoming connections over **multiple** listeners
// * dialing outgoing connections over **multiple** transports
// * multiplexing **multiple** connections per-peer
// * multiplexing **multiple** different servers or protocols
// * handling backpressure correctly
// * handling stream multiplexing
// * providing a **simple** interface to the user
package peerstream
