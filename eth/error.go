package eth

import (
	"fmt"
)

const (
	ErrMsgTooLarge = iota
	ErrDecode
	ErrInvalidMsgCode
	ErrProtocolVersionMismatch
	ErrNetworkIdMismatch
	ErrGenesisBlockMismatch
	ErrNoStatusMsg
	ErrExtraStatusMsg
	ErrInvalidBlock
	ErrInvalidPoW
	ErrUnrequestedBlock
)

var errorToString = map[int]string{
	ErrMsgTooLarge:             "Message too long",
	ErrDecode:                  "Invalid message",
	ErrInvalidMsgCode:          "Invalid message code",
	ErrProtocolVersionMismatch: "Protocol version mismatch",
	ErrNetworkIdMismatch:       "NetworkId mismatch",
	ErrGenesisBlockMismatch:    "Genesis block mismatch",
	ErrNoStatusMsg:             "No status message",
	ErrExtraStatusMsg:          "Extra status message",
	ErrInvalidBlock:            "Invalid block",
	ErrInvalidPoW:              "Invalid PoW",
	ErrUnrequestedBlock:        "Unrequested block",
}

type protocolError struct {
	Code    int
	fatal   bool
	message string
	format  string
	params  []interface{}
	// size    int
}

func newProtocolError(code int, format string, params ...interface{}) *protocolError {
	return &protocolError{Code: code, format: format, params: params}
}

func ProtocolError(code int, format string, params ...interface{}) (err *protocolError) {
	err = newProtocolError(code, format, params...)
	// report(err)
	return
}

func (self protocolError) Error() (message string) {
	if len(message) == 0 {
		var ok bool
		self.message, ok = errorToString[self.Code]
		if !ok {
			panic("invalid error code")
		}
		if self.format != "" {
			self.message += ": " + fmt.Sprintf(self.format, self.params...)
		}
	}
	return self.message
}

func (self *protocolError) Fatal() bool {
	return self.fatal
}
