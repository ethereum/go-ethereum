// Package secio is used to encrypt `go-libp2p-conn` connections. Connections wrapped by secio use secure sessions provided by this package to encrypt all traffic. A TLS-like handshake is used to setup the communication channel.
package secio

import (
	"context"
	"io"

	ci "github.com/libp2p/go-libp2p-crypto"
	peer "github.com/libp2p/go-libp2p-peer"
	msgio "github.com/libp2p/go-msgio"
)

// SessionGenerator constructs secure communication sessions for a peer.
type SessionGenerator struct {
	LocalID    peer.ID
	PrivateKey ci.PrivKey
}

// NewSession takes an insecure io.ReadWriter, performs a TLS-like
// handshake with the other side, and returns a secure session.
// See the source for the protocol details and security implementation.
func (sg *SessionGenerator) NewSession(ctx context.Context, insecure io.ReadWriteCloser) (Session, error) {
	return newSecureSession(ctx, sg.LocalID, sg.PrivateKey, insecure)
}

// Session provides the necessary functionality to wrap a connection
// and tunnel it through a secure channel via the provided ReadWriter.
type Session interface {
	// ReadWriter returns the encrypted communication channel
	ReadWriter() msgio.ReadWriteCloser

	// LocalPeer retrieves the local peer.
	LocalPeer() peer.ID

	// LocalPrivateKey retrieves the local private key
	LocalPrivateKey() ci.PrivKey

	// RemotePeer retrieves the remote peer.
	RemotePeer() peer.ID

	// RemotePublicKey retrieves the remote's public key
	// which was received during the handshake.
	RemotePublicKey() ci.PubKey

	// Close closes the secure session
	Close() error
}

// ReadWriter returns the encrypted communication channel
func (s *secureSession) ReadWriter() msgio.ReadWriteCloser {
	return s.secure
}

// LocalPeer retrieves the local peer.
func (s *secureSession) LocalPeer() peer.ID {
	return s.localPeer
}

// LocalPrivateKey retrieves the local peer's PrivateKey
func (s *secureSession) LocalPrivateKey() ci.PrivKey {
	return s.localKey
}

// RemotePeer retrieves the remote peer.
func (s *secureSession) RemotePeer() peer.ID {
	return s.remotePeer
}

// RemotePublicKey retrieves the remote public key.
func (s *secureSession) RemotePublicKey() ci.PubKey {
	return s.remote.permanentPubKey
}

// Close closes the secure session
func (s *secureSession) Close() error {
	return s.secure.Close()
}
