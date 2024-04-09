package discover

import (
	"fmt"
	"github.com/ethereum/go-ethereum/p2p/discover/v4wire"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/google/gofuzz"
	"net"
	"testing"
	"time"
)

func TestMutatePingMsg(t *testing.T) {
	// 创建一个 Ping 消息
	ping := &v4wire.Ping{
		Version:    1,
		From:       v4wire.Endpoint{IP: net.IP("127.0.0.1"), UDP: 30303, TCP: 30303},
		To:         v4wire.Endpoint{IP: net.IP("127.0.0.2"), UDP: 30303, TCP: 30303},
		Expiration: 1234567890,
		ENRSeq:     42,
		Rest:       []rlp.RawValue{[]byte{0x01, 0x02, 0x03}},
	}

	// 使用 gofuzz 创建一个 fuzzer
	f := fuzz.New().NilChance(0.1)

	fmt.Println("Version: ", ping.Version)
	// 对 Ping 消息进行变异
	MutatePingMsg(f, ping)
	fmt.Println("Version: ", ping.Version)

	// 在这里进行断言或其他需要的验证操作
	// 例如，检查变异后的 ping.Version 是否发生了变化
	if ping.Version == 1 {
		t.Error("Ping message version did not mutate")
	}
}

func TestFuzzMsgs(t *testing.T) {
	// 创建一个 UDPv4 实例
	cfg := &Config{}
	cfg.withDefaults()

	udp := &UDPv4{
		reqSend:    make(chan v4wire.Packet),
		reqReceive: make(chan v4wire.Packet),
		log:        cfg.Log,
	}

	// 启动模拟发送和变异消息的 goroutine
	go udp.FuzzMsgs()

	// 创建一个 Ping 消息
	ping := &v4wire.Ping{
		Version:    1,
		From:       v4wire.Endpoint{IP: net.IP("127.0.0.1"), UDP: 30303, TCP: 30303},
		To:         v4wire.Endpoint{IP: net.IP("127.0.0.2"), UDP: 30303, TCP: 30303},
		Expiration: 1234567890,
		ENRSeq:     42,
		Rest:       []rlp.RawValue{[]byte{0x01, 0x02, 0x03}},
	}

	// 将 Ping 消息发送到 reqSend 通道
	udp.reqSend <- ping

	// 等待一段时间以确保消息被处理
	time.Sleep(time.Second)

	// 从 reqReceive 通道接收变异后的消息
	receivedMsg := <-udp.reqReceive

	// 检查接收到的消息是否是变异后的 Ping 消息
	receivedPing, ok := receivedMsg.(*v4wire.Ping)
	if !ok {
		t.Error("Received message is not of type *v4wire.Ping")
	}

	// 在这里进行断言或其他需要的验证操作
	// 例如，检查变异后的 ping.Version 是否发生了变化
	if receivedPing.Version == 1 {
		t.Error("Ping message version did not mutate")
	}
}
