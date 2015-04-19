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
	errPubkeyInvalid
	errPubkeyForbidden
	errProtocolBreach
	errPingTimeout
	errInvalidNetworkId
	errInvalidProtocolVersion
)

var errorToString = map[int]string{
	errMagicTokenMismatch:     "magic token mismatch",
	errRead:                   "read error",
	errWrite:                  "write error",
	errMisc:                   "misc error",
	errInvalidMsgCode:         "invalid message code",
	errInvalidMsg:             "invalid message",
	errP2PVersionMismatch:     "P2P Version Mismatch",
	errPubkeyInvalid:          "public key invalid",
	errPubkeyForbidden:        "public key forbidden",
	errProtocolBreach:         "protocol Breach",
	errPingTimeout:            "ping timeout",
	errInvalidNetworkId:       "invalid network id",
	errInvalidProtocolVersion: "invalid protocol version",
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

type DiscReason uint

const (
	DiscRequested DiscReason = iota
	DiscNetworkError
	DiscProtocolError
	DiscUselessPeer
	DiscTooManyPeers
	DiscAlreadyConnected
	DiscIncompatibleVersion
	DiscInvalidIdentity
	DiscQuitting
	DiscUnexpectedIdentity
	DiscSelf
	DiscReadTimeout
	DiscSubprotocolError
)

var discReasonToString = [...]string{
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

func (d DiscReason) Error() string {
	return d.String()
}

func discReasonForError(err error) DiscReason {
	if reason, ok := err.(DiscReason); ok {
		return reason
	}
	peerError, ok := err.(*peerError)
	if !ok {
		return DiscSubprotocolError
	}
	switch peerError.Code {
	case errP2PVersionMismatch:
		return DiscIncompatibleVersion
	case errPubkeyInvalid:
		return DiscInvalidIdentity
	case errPubkeyForbidden:
		return DiscUselessPeer
	case errInvalidMsgCode, errMagicTokenMismatch, errProtocolBreach:
		return DiscProtocolError
	case errPingTimeout:
		return DiscReadTimeout
	case errRead, errWrite:
		return DiscNetworkError
	default:
		return DiscSubprotocolError
	}
}
