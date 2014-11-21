package p2p

import (
	"fmt"
)

const (
	errMagicTokenMismatch = iota
	errRead
	errWrite
	errMisc
	errInvalidMsgCode
	errInvalidMsg
	errP2PVersionMismatch
	errPubkeyMissing
	errPubkeyInvalid
	errPubkeyForbidden
	errProtocolBreach
	errPingTimeout
	errInvalidNetworkId
	errInvalidProtocolVersion
)

var errorToString = map[int]string{
	errMagicTokenMismatch:     "Magic token mismatch",
	errRead:                   "Read error",
	errWrite:                  "Write error",
	errMisc:                   "Misc error",
	errInvalidMsgCode:         "Invalid message code",
	errInvalidMsg:             "Invalid message",
	errP2PVersionMismatch:     "P2P Version Mismatch",
	errPubkeyMissing:          "Public key missing",
	errPubkeyInvalid:          "Public key invalid",
	errPubkeyForbidden:        "Public key forbidden",
	errProtocolBreach:         "Protocol Breach",
	errPingTimeout:            "Ping timeout",
	errInvalidNetworkId:       "Invalid network id",
	errInvalidProtocolVersion: "Invalid protocol version",
}

type peerError struct {
	Code    int
	message string
}

func newPeerError(code int, format string, v ...interface{}) *peerError {
	desc, ok := errorToString[code]
	if !ok {
		panic("invalid error code")
	}
	err := &peerError{code, desc}
	if format != "" {
		err.message += ": " + fmt.Sprintf(format, v...)
	}
	return err
}

func (self *peerError) Error() string {
	return self.message
}

type DiscReason byte

const (
	DiscRequested           DiscReason = 0x00
	DiscNetworkError                   = 0x01
	DiscProtocolError                  = 0x02
	DiscUselessPeer                    = 0x03
	DiscTooManyPeers                   = 0x04
	DiscAlreadyConnected               = 0x05
	DiscIncompatibleVersion            = 0x06
	DiscInvalidIdentity                = 0x07
	DiscQuitting                       = 0x08
	DiscUnexpectedIdentity             = 0x09
	DiscSelf                           = 0x0a
	DiscReadTimeout                    = 0x0b
	DiscSubprotocolError               = 0x10
)

var discReasonToString = [DiscSubprotocolError + 1]string{
	DiscRequested:           "Disconnect requested",
	DiscNetworkError:        "Network error",
	DiscProtocolError:       "Breach of protocol",
	DiscUselessPeer:         "Useless peer",
	DiscTooManyPeers:        "Too many peers",
	DiscAlreadyConnected:    "Already connected",
	DiscIncompatibleVersion: "Incompatible P2P protocol version",
	DiscInvalidIdentity:     "Invalid node identity",
	DiscQuitting:            "Client quitting",
	DiscUnexpectedIdentity:  "Unexpected identity",
	DiscSelf:                "Connected to self",
	DiscReadTimeout:         "Read timeout",
	DiscSubprotocolError:    "Subprotocol error",
}

func (d DiscReason) String() string {
	if len(discReasonToString) < int(d) {
		return fmt.Sprintf("Unknown Reason(%d)", d)
	}
	return discReasonToString[d]
}

func discReasonForError(err error) DiscReason {
	peerError, ok := err.(*peerError)
	if !ok {
		return DiscSubprotocolError
	}
	switch peerError.Code {
	case errP2PVersionMismatch:
		return DiscIncompatibleVersion
	case errPubkeyMissing, errPubkeyInvalid:
		return DiscInvalidIdentity
	case errPubkeyForbidden:
		return DiscUselessPeer
	case errInvalidMsgCode, errMagicTokenMismatch, errProtocolBreach:
		return DiscProtocolError
	case errPingTimeout:
		return DiscReadTimeout
	case errRead, errWrite, errMisc:
		return DiscNetworkError
	default:
		return DiscSubprotocolError
	}
}
