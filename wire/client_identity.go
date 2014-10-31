package wire

import (
	"fmt"
	"runtime"
)

// should be used in Peer handleHandshake, incorporate Caps, ProtocolVersion, Pubkey etc.
type ClientIdentity interface {
	String() string
}

type SimpleClientIdentity struct {
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
		implementation:   runtime.Version(),
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

func (c *SimpleClientIdentity) SetCustomIdentifier(customIdentifier string) {
	c.customIdentifier = customIdentifier
}

func (c *SimpleClientIdentity) GetCustomIdentifier() string {
	return c.customIdentifier
}
