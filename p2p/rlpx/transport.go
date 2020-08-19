package rlpx

import "crypto/ecdsa"

type Transport interface {
	// The two handshakes.
	DoEncHandshake(prv *ecdsa.PrivateKey, dialDest *ecdsa.PublicKey) (*ecdsa.PublicKey, error)
	DoProtoHandshake(our *protoHandshake) (*protoHandshake, error)

	MsgREadWriter

	// TODO how to do this? we need a msg read writer in rlpx but how should it look?
	ReadMsg() (RawRLPXMessage, error)
	WriteMsg(RawRLPXMessage) error

	// transports must provide Close because we use MsgPipe in some of
	// the tests. Closing the actual network connection doesn't do
	// anything in those tests because MsgPipe doesn't use it.
	Close(err error)
}
