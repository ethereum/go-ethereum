package qwhisper

import (
	"time"

	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/whisper"
)

func fromHex(s string) []byte {
	if len(s) > 1 {
		return ethutil.Hex2Bytes(s[2:])
	}
	return nil
}
func toHex(b []byte) string { return "0x" + ethutil.Bytes2Hex(b) }

type Whisper struct {
	*whisper.Whisper
}

func New(w *whisper.Whisper) *Whisper {
	return &Whisper{w}
}

func (self *Whisper) Post(data string, pow, ttl uint32, to, from string) {
	msg := whisper.NewMessage(fromHex(data))
	envelope, err := msg.Seal(time.Duration(pow), whisper.Opts{
		Ttl:  time.Duration(ttl),
		To:   crypto.PubTECDSA(fromHex(to)),
		From: crypto.ToECDSA(fromHex(from)),
	})
	if err != nil {
		// handle error
		return
	}

	if err := self.Whisper.Send(envolpe); err != nil {
		// handle error
		return
	}
}

func (self *Whisper) NewIdentity() string {
	return toHex(self.Whisper.NewIdentity().D.Bytes())
}

func (self *Whisper) HasIdentify(key string) bool {
	return self.Whisper.HasIdentity(crypto.ToECDSA(fromHex(key)))
}

func (self *Whisper) Watch(opts map[string]interface{}) {
	filter := filterFromMap(opts)
	filter.Fn = func(msg *Message) {
		// TODO POST TO QT WINDOW
	}
	self.Watch(filter)
}

func filterFromMap(opts map[string]interface{}) whisper.Filter {
	var f Filter
	if to, ok := opts["to"].(string); ok {
		f.To = ToECDSA(fromHex(to))
	}
	if from, ok := opts["from"].(string); ok {
		f.From = ToECDSAPub(fromHex(from))
	}

	return f
}
