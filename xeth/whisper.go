package xeth

import (
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/whisper"
)

var qlogger = logger.NewLogger("XSHH")

type Whisper struct {
	*whisper.Whisper
}

func NewWhisper(w *whisper.Whisper) *Whisper {
	return &Whisper{w}
}

func (self *Whisper) Post(payload string, to, from string, topics []string, priority, ttl uint32) error {
	if priority == 0 {
		priority = 1000
	}

	if ttl == 0 {
		ttl = 100
	}

	pk := crypto.ToECDSAPub(fromHex(from))
	if key := self.Whisper.GetIdentity(pk); key != nil || len(from) == 0 {
		msg := whisper.NewMessage(fromHex(payload))
		envelope, err := msg.Seal(time.Duration(priority*100000), whisper.Opts{
			Ttl:    time.Duration(ttl) * time.Second,
			To:     crypto.ToECDSAPub(fromHex(to)),
			From:   key,
			Topics: whisper.TopicsFromString(topics...),
		})

		if err != nil {
			return err
		}

		if err := self.Whisper.Send(envelope); err != nil {
			return err
		}
	} else {
		return errors.New("unmatched pub / priv for seal")
	}

	return nil
}

func (self *Whisper) NewIdentity() string {
	key := self.Whisper.NewIdentity()

	return toHex(crypto.FromECDSAPub(&key.PublicKey))
}

func (self *Whisper) HasIdentity(key string) bool {
	return self.Whisper.HasIdentity(crypto.ToECDSAPub(fromHex(key)))
}

func (self *Whisper) Watch(opts *Options) int {
	filter := whisper.Filter{
		To:     crypto.ToECDSAPub(fromHex(opts.To)),
		From:   crypto.ToECDSAPub(fromHex(opts.From)),
		Topics: whisper.TopicsFromString(opts.Topics...),
	}

	var i int
	filter.Fn = func(msg *whisper.Message) {
		opts.Fn(NewWhisperMessage(msg))
	}

	i = self.Whisper.Watch(filter)

	return i
}

func (self *Whisper) Messages(id int) (messages []WhisperMessage) {
	msgs := self.Whisper.Messages(id)
	messages = make([]WhisperMessage, len(msgs))
	for i, message := range msgs {
		messages[i] = NewWhisperMessage(message)
	}

	return
}

type Options struct {
	To     string
	From   string
	Topics []string
	Fn     func(msg WhisperMessage)
}

type WhisperMessage struct {
	ref     *whisper.Message
	Payload string `json:"payload"`
	To      string `json:"to"`
	From    string `json:"from"`
	Sent    int64  `json:"sent"`
}

func NewWhisperMessage(msg *whisper.Message) WhisperMessage {
	return WhisperMessage{
		ref:     msg,
		Payload: toHex(msg.Payload),
		From:    toHex(crypto.FromECDSAPub(msg.Recover())),
		To:      toHex(crypto.FromECDSAPub(msg.To)),
		Sent:    msg.Sent,
	}
}
