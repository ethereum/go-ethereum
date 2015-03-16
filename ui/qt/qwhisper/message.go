package qwhisper

import (
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/whisper"
)

type Message struct {
	ref     *whisper.Message
	Flags   int32  `json:"flags"`
	Payload string `json:"payload"`
	From    string `json:"from"`
}

func ToQMessage(msg *whisper.Message) *Message {
	return &Message{
		ref:     msg,
		Flags:   int32(msg.Flags),
		Payload: "0x" + common.Bytes2Hex(msg.Payload),
		From:    "0x" + common.Bytes2Hex(crypto.FromECDSAPub(msg.Recover())),
	}
}
