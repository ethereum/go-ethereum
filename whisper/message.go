package whisper

type Message struct {
	Flags     byte
	Signature []byte
	Payload   []byte
}

func NewMessage(payload []byte) *Message {
	return &Message{Flags: 0, Payload: payload}
}

func (self *Message) Bytes() []byte {
	return append([]byte{self.Flags}, append(self.Signature, self.Payload...)...)
}
