package whisper

import (
	"fmt"
	"testing"
	"time"
)

func TestKeyManagement(t *testing.T) {
	whisper := New()

	key := whisper.NewIdentity()
	if !whisper.HasIdentity(key) {
		t.Error("expected whisper to have identify")
	}
}

func TestEvent(t *testing.T) {
	res := make(chan *Message, 1)
	whisper := New()
	id := whisper.NewIdentity()
	whisper.Watch(Filter{
		To: id,
		Fn: func(msg *Message) {
			res <- msg
		},
	})

	msg := NewMessage([]byte(fmt.Sprintf("Hello world. This is whisper-go. Incase you're wondering; the time is %v", time.Now())))
	envelope, err := msg.Seal(DefaultPow, Opts{
		Ttl:  DefaultTtl,
		From: id,
		To:   &id.PublicKey,
	})
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}

	tick := time.NewTicker(time.Second)
	whisper.postEvent(envelope)
	select {
	case <-res:
	case <-tick.C:
		t.Error("did not receive message")
	}
}
