package discover

import (
	"github.com/ethereum/go-ethereum/p2p/discover/v4wire"
	fuzz "github.com/google/gofuzz"
	"time"
)

var fuzzcount = 0
var times = 0

func (t *UDPv4) FuzzMsgs() {
	//TODO temp: send mutated tx here!

	time.Sleep(time.Duration(2) * time.Second) //sleep 2 minutes

	//t.Log().Error("Begin sending Fuzzed Transactions!!!!!!!")

	f := fuzz.New().NilChance(0.1)

	//for {
	//	//p.Log().Warn("Sending Fuzzed Message!")
	//	MsgFuzzed(t, f)
	//	time.Sleep(time.Duration(1) * time.Second)
	//}
	//t.log.Warn("Start Fuzzing!!!")

	for {
		select {
		case req := <-t.reqSend:
			switch r := req.(type) {
			case *v4wire.Ping:
				// 处理 Ping 类型的 req
				// 例如：
				// 修改 r 的字段
				// 将修改后的 r 发送给 reqReceive
				// t.reqReceive <- r
				MutatePingMsg(f, r)
				t.reqReceive <- r
			case *v4wire.Pong:
				MutatePongMsg(f, r)
				t.reqReceive <- r
			case *v4wire.Findnode:
				MutateFindnodeMsg(f, r)
				t.reqReceive <- r
			case *v4wire.Neighbors:
				MutateNeighborsMsg(f, r)
			case *v4wire.ENRRequest:
				MutateENRRequestMsg(f, r)
				t.reqReceive <- r
			case *v4wire.ENRResponse:
				MutateENRResponseMsg(f, r)
				t.reqReceive <- r
			default:
				select {}
			}
		}
	}
}

func MutatePingMsg(f *fuzz.Fuzzer, msg *v4wire.Ping) {
	f.Fuzz(&msg.Version)
}

func MutatePongMsg(f *fuzz.Fuzzer, msg *v4wire.Pong) {
	f.Fuzz(msg.To)
}

func MutateFindnodeMsg(f *fuzz.Fuzzer, msg *v4wire.Findnode) {
	f.Fuzz(msg.Expiration)
}

func MutateNeighborsMsg(f *fuzz.Fuzzer, msg *v4wire.Neighbors) {
	f.Fuzz(msg.Expiration)
}

func MutateENRRequestMsg(f *fuzz.Fuzzer, msg *v4wire.ENRRequest) {
	f.Fuzz(msg.Expiration)
}

func MutateENRResponseMsg(f *fuzz.Fuzzer, msg *v4wire.ENRResponse) {
	f.Fuzz(msg.Record)
}
