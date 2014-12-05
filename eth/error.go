package eth

import (
	"fmt"
	// "github.com/ethereum/go-ethereum/logger"
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
	if err.Fatal() {
		logger.Errorln(err)
	} else {
		logger.Debugln(err)
	}
	return
}

func (self protocolError) Error() (message string) {
	message = self.message
	if message == "" {
		message, ok := errorToString[self.Code]
		if !ok {
			panic("invalid error code")
		}
		if self.format != "" {
			message += ": " + fmt.Sprintf(self.format, self.params...)
		}
		self.message = message
	}
	return
}

func (self *protocolError) Fatal() bool {
	return self.fatal
}
