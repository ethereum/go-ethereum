// Contains the external API representation of a whisper message.

package xeth

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/whisper"
)

// WhisperMessage is the external API representation of a whisper.Message.
type WhisperMessage struct {
	ref *whisper.Message

	Payload string `json:"payload"`
	To      string `json:"to"`
	From    string `json:"from"`
	Sent    int64  `json:"sent"`
}

// NewWhisperMessage converts an internal message into an API version.
func NewWhisperMessage(message *whisper.Message) WhisperMessage {
	return WhisperMessage{
		ref: message,

		Payload: common.ToHex(message.Payload),
		From:    common.ToHex(crypto.FromECDSAPub(message.Recover())),
		To:      common.ToHex(crypto.FromECDSAPub(message.To)),
		Sent:    message.Sent,
	}
}
