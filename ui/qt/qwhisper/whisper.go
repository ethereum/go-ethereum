// QWhisper package. This package is temporarily on hold until QML DApp dev will reemerge.
package qwhisper

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/whisper"
	"github.com/obscuren/qml"
)

var qlogger = logger.NewLogger("QSHH")

type Whisper struct {
	*whisper.Whisper
	view qml.Object

	watches map[int]*Watch
}

func New(w *whisper.Whisper) *Whisper {
	return &Whisper{w, nil, make(map[int]*Watch)}
}

func (self *Whisper) SetView(view qml.Object) {
	self.view = view
}

func (self *Whisper) Post(payload []string, to, from string, topics []string, priority, ttl uint32) {
	var data []byte
	for _, d := range payload {
		data = append(data, common.FromHex(d)...)
	}

	pk := crypto.ToECDSAPub(common.FromHex(from))
	if key := self.Whisper.GetIdentity(pk); key != nil {
		msg := whisper.NewMessage(data)
		envelope, err := msg.Wrap(time.Duration(priority*100000), whisper.Options{
			TTL:    time.Duration(ttl) * time.Second,
			To:     crypto.ToECDSAPub(common.FromHex(to)),
			From:   key,
			Topics: whisper.NewTopicsFromStrings(topics...),
		})

		if err != nil {
			qlogger.Infoln(err)
			// handle error
			return
		}

		if err := self.Whisper.Send(envelope); err != nil {
			qlogger.Infoln(err)
			// handle error
			return
		}
	} else {
		qlogger.Infoln("unmatched pub / priv for seal")
	}

}

func (self *Whisper) NewIdentity() string {
	key := self.Whisper.NewIdentity()

	return common.ToHex(crypto.FromECDSAPub(&key.PublicKey))
}

func (self *Whisper) HasIdentity(key string) bool {
	return self.Whisper.HasIdentity(crypto.ToECDSAPub(common.FromHex(key)))
}

func (self *Whisper) Watch(opts map[string]interface{}, view *qml.Common) int {
	filter := filterFromMap(opts)
	var i int
	filter.Fn = func(msg *whisper.Message) {
		if view != nil {
			view.Call("onShhMessage", ToQMessage(msg), i)
		}
	}

	i = self.Whisper.Watch(filter)
	self.watches[i] = &Watch{}

	return i
}

func (self *Whisper) Messages(id int) (messages *common.List) {
	msgs := self.Whisper.Messages(id)
	messages = common.EmptyList()
	for _, message := range msgs {
		messages.Append(ToQMessage(message))
	}

	return
}

func filterFromMap(opts map[string]interface{}) (f whisper.Filter) {
	if to, ok := opts["to"].(string); ok {
		f.To = crypto.ToECDSAPub(common.FromHex(to))
	}
	if from, ok := opts["from"].(string); ok {
		f.From = crypto.ToECDSAPub(common.FromHex(from))
	}
	if topicList, ok := opts["topics"].(*qml.List); ok {
		var topics []string
		topicList.Convert(&topics)
		f.Topics = whisper.NewFilterTopicsFromStringsFlat(topics...)
	}

	return
}
