package p2p

import (
	"fmt"
)

type ErrorCode int

const errorChanCapacity = 10

const (
	PacketTooShort = iota
	PayloadTooShort
	MagicTokenMismatch
	EmptyPayload
	ReadError
	WriteError
	MiscError
	InvalidMsgCode
	InvalidMsg
	P2PVersionMismatch
	PubkeyMissing
	PubkeyInvalid
	PubkeyForbidden
	ProtocolBreach
	PortMismatch
	PingTimeout
	InvalidGenesis
	InvalidNetworkId
	InvalidProtocolVersion
)

var errorToString = map[ErrorCode]string{
	PacketTooShort:         "Packet too short",
	PayloadTooShort:        "Payload too short",
	MagicTokenMismatch:     "Magic token mismatch",
	EmptyPayload:           "Empty payload",
	ReadError:              "Read error",
	WriteError:             "Write error",
	MiscError:              "Misc error",
	InvalidMsgCode:         "Invalid message code",
	InvalidMsg:             "Invalid message",
	P2PVersionMismatch:     "P2P Version Mismatch",
	PubkeyMissing:          "Public key missing",
	PubkeyInvalid:          "Public key invalid",
	PubkeyForbidden:        "Public key forbidden",
	ProtocolBreach:         "Protocol Breach",
	PortMismatch:           "Port mismatch",
	PingTimeout:            "Ping timeout",
	InvalidGenesis:         "Invalid genesis block",
	InvalidNetworkId:       "Invalid network id",
	InvalidProtocolVersion: "Invalid protocol version",
}

type PeerError struct {
	Code    ErrorCode
	message string
}

func NewPeerError(code ErrorCode, format string, v ...interface{}) *PeerError {
	desc, ok := errorToString[code]
	if !ok {
		panic("invalid error code")
	}
	format = desc + ": " + format
	message := fmt.Sprintf(format, v...)
	return &PeerError{code, message}
}

func (self *PeerError) Error() string {
	return self.message
}

func NewPeerErrorChannel() chan *PeerError {
	return make(chan *PeerError, errorChanCapacity)
}
