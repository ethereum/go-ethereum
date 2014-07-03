package ethwire

import (
	"fmt"
	"runtime"
)

// should be used in Peer handleHandshake, incorporate Caps, ProtocolVersion, Pubkey etc.
type ClientIdentity interface {
	String() string
}

type SimpleClientIdentity struct {
	clientString     string
	clientIdentifier string
	version          string
	customIdentifier string
	os               string
	implementation   string
}

func NewSimpleClientIdentity(clientIdentifier string, version string, customIdentifier string) *SimpleClientIdentity {
	clientIdentity := &SimpleClientIdentity{
		clientIdentifier: clientIdentifier,
		version:          version,
		customIdentifier: customIdentifier,
		os:               runtime.GOOS,
		implementation:   "Go",
	}
	clientIdentity.init()
	return clientIdentity
}

func (c *SimpleClientIdentity) init() {
	c.clientString = fmt.Sprintf("%s/v%s/%s/%s/%s",
		c.clientIdentifier,
		c.version,
		c.customIdentifier,
		c.os,
		c.implementation)
}

func (c *SimpleClientIdentity) String() string {
	return c.clientString
}

func (c *SimpleClientIdentity) SetCustomIdentifier(customIdentifier string) {
	c.customIdentifier = customIdentifier
	c.init()
}

func (c *SimpleClientIdentity) GetCustomIdentifier() string {
	return c.customIdentifier
}
