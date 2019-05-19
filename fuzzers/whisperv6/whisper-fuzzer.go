package whisperv6

import (
	"bytes"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/whisper/whisperv6"
)

type MessageParams struct {
	Topic    whisperv6.TopicType
	WorkTime uint32
	TTL      uint32
	KeySym   []byte
	Payload  []byte
}

//export fuzzer_entry
func Fuzz(input []byte) int {

	var paramsDecoded MessageParams
	err := rlp.DecodeBytes(input, &paramsDecoded)
	if err != nil {
		return 0
	}
	var params whisperv6.MessageParams
	params.KeySym = make([]byte, 32)
	if len(paramsDecoded.KeySym) <= 32 {
		copy(params.KeySym, paramsDecoded.KeySym)
	}
	if input[0] == 255 {
		params.PoW = 0.01
		params.WorkTime = 1
	} else {
		params.PoW = 0
		params.WorkTime = 0
	}
	params.TTL = paramsDecoded.TTL
	params.Payload = paramsDecoded.Payload
	text := make([]byte, 0, 512)
	text = append(text, params.Payload...)
	params.Topic = paramsDecoded.Topic
	params.Src, err = crypto.GenerateKey()
	if err != nil {
		return 0
	}
	msg, err := whisperv6.NewSentMessage(&params)
	if err != nil {
		panic(err)
		//return
	}
	env, err := msg.Wrap(&params)
	if err != nil {
		panic(err)
	}
	decrypted, err := env.OpenSymmetric(params.KeySym)
	if err != nil {
		panic(err)
	}
	if !decrypted.ValidateAndParse() {
		panic("ValidateAndParse failed")
		return 0
	}
	if !bytes.Equal(text, decrypted.Payload) {
		panic("text != decrypted.Payload")
	}
	if len(decrypted.Signature) != 65 {
		panic("Unexpected signature length")
	}
	if !whisperv6.IsPubKeyEqual(decrypted.Src, &params.Src.PublicKey) {
		panic("Unexpected public key")
	}

	return 0
}
