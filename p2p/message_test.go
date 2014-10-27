package p2p

import (
	"testing"
)

func TestNewMsg(t *testing.T) {
	msg, _ := NewMsg(3, 1, "000")
	if msg.Code() != 3 {
		t.Errorf("incorrect code %v", msg.Code())
	}
	data0 := msg.Data().Get(0).Uint()
	data1 := string(msg.Data().Get(1).Bytes())
	if data0 != 1 {
		t.Errorf("incorrect data %v", data0)
	}
	if data1 != "000" {
		t.Errorf("incorrect data %v", data1)
	}
}

func TestEncodeDecodeMsg(t *testing.T) {
	msg, _ := NewMsg(3, 1, "000")
	encoded := msg.Encode(3)
	msg, _ = NewMsgFromBytes(encoded)
	msg.Decode(3)
	if msg.Code() != 3 {
		t.Errorf("incorrect code %v", msg.Code())
	}
	data0 := msg.Data().Get(0).Uint()
	data1 := msg.Data().Get(1).Str()
	if data0 != 1 {
		t.Errorf("incorrect data %v", data0)
	}
	if data1 != "000" {
		t.Errorf("incorrect data %v", data1)
	}
}
