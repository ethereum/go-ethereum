package p2p

import (
	"fmt"
	"runtime"
)

// ClientIdentity represents the identity of a peer.
type ClientIdentity interface {
	String() string // human readable identity
	Pubkey() []byte // 512-bit public key
}

type SimpleClientIdentity struct {
	clientIdentifier string
	version          string
	customIdentifier string
	os               string
	implementation   string
	pubkey           string
}

func NewSimpleClientIdentity(clientIdentifier string, version string, customIdentifier string, pubkey string) *SimpleClientIdentity {
	clientIdentity := &SimpleClientIdentity{
		clientIdentifier: clientIdentifier,
		version:          version,
		customIdentifier: customIdentifier,
		os:               runtime.GOOS,
		implementation:   runtime.Version(),
		pubkey:           pubkey,
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

func (c *SimpleClientIdentity) Pubkey() []byte {
	return []byte(c.pubkey)
}

func (c *SimpleClientIdentity) SetCustomIdentifier(customIdentifier string) {
	c.customIdentifier = customIdentifier
}

func (c *SimpleClientIdentity) GetCustomIdentifier() string {
	return c.customIdentifier
}
