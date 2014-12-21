package qwhisper

import (
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/whisper"
)

type Message struct {
	ref     *whisper.Message
	Flags   byte
	Payload string
	From    string
}

func ToQMessage(msg *whisper.Message) *Message {
	return &Message{
		ref:     msg,
		Flags:   msg.Flags,
		Payload: ethutil.Bytes2Hex(msg.Payload),
		From:    ethutil.Bytes2Hex(crypto.FromECDSAPub(msg.Recover())),
	}
}
