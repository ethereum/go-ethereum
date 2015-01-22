package p2p

import (
	"fmt"
	"runtime"
)

// ClientIdentity represents the identity of a peer.
type ClientIdentity interface {
	String() string     // human readable identity
	PublicKey() []byte  // 512-bit public key represented in 65 byte format as per golang/elliptic.Marshal, first byte encodes curve
	PrivateKey() []byte // 256-bit private key
}

type SimpleClientIdentity struct {
	clientIdentifier string
	version          string
	customIdentifier string
	os               string
	implementation   string
	privkey          []byte
	pubkey           []byte
}

func NewSimpleClientIdentity(clientIdentifier string, version string, customIdentifier string, privkey []byte, pubkey []byte) *SimpleClientIdentity {
	clientIdentity := &SimpleClientIdentity{
		clientIdentifier: clientIdentifier,
		version:          version,
		customIdentifier: customIdentifier,
		os:               runtime.GOOS,
		implementation:   runtime.Version(),
		pubkey:           pubkey,
		privkey:          privkey,
	}

	return clientIdentity
}

func (c *SimpleClientIdentity) init() {
}

func (c *SimpleClientIdentity) String() string {
	var id string
	if len(c.customIdentifier) > 0 {
		id = "/" + c.customIdentifier
	}

	return fmt.Sprintf("%s/v%s%s/%s/%s",
		c.clientIdentifier,
		c.version,
		id,
		c.os,
		c.implementation)
}

func (c *SimpleClientIdentity) PrivateKey() []byte {
	return c.privkey
}

func (c *SimpleClientIdentity) PublicKey() []byte {
	return c.pubkey
}

func (c *SimpleClientIdentity) SetCustomIdentifier(customIdentifier string) {
	c.customIdentifier = customIdentifier
}

func (c *SimpleClientIdentity) GetCustomIdentifier() string {
	return c.customIdentifier
}
