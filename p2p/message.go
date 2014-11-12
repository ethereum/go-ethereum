package p2p

import (
	// "fmt"
	"github.com/ethereum/go-ethereum/ethutil"
)

type MsgCode uint8

type Msg struct {
	code    MsgCode // this is the raw code as per adaptive msg code scheme
	data    *ethutil.Value
	encoded []byte
}

func (self *Msg) Code() MsgCode {
	return self.code
}

func (self *Msg) Data() *ethutil.Value {
	return self.data
}

func NewMsg(code MsgCode, params ...interface{}) (msg *Msg, err error) {

	// // data := [][]interface{}{}
	// data := []interface{}{}
	// for _, value := range params {
	// 	if encodable, ok := value.(ethutil.RlpEncodeDecode); ok {
	// 		data = append(data, encodable.RlpValue())
	// 	} else if raw, ok := value.([]interface{}); ok {
	// 		data = append(data, raw)
	// 	} else {
	// 		// data = append(data, interface{}(raw))
	// 		err = fmt.Errorf("Unable to encode object of type %T", value)
	// 		return
	// 	}
	// }
	return &Msg{
		code: code,
		data: ethutil.NewValue(interface{}(params)),
	}, nil
}

func NewMsgFromBytes(encoded []byte) (msg *Msg, err error) {
	value := ethutil.NewValueFromBytes(encoded)
	// Type of message
	code := value.Get(0).Uint()
	// Actual data
	data := value.SliceFrom(1)

	msg = &Msg{
		code: MsgCode(code),
		data: data,
		// data:    ethutil.NewValue(data),
		encoded: encoded,
	}
	return
}

func (self *Msg) Decode(offset MsgCode) {
	self.code = self.code - offset
}

// encode takes an offset argument to implement adaptive message coding
// the encoded message is memoized to make msgs relayed to several peers more efficient
func (self *Msg) Encode(offset MsgCode) (res []byte) {
	if len(self.encoded) == 0 {
		res = ethutil.NewValue(append([]interface{}{byte(self.code + offset)}, self.data.Slice()...)).Encode()
		self.encoded = res
	} else {
		res = self.encoded
	}
	return
}
