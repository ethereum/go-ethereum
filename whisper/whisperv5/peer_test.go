// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package whisperv5

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/nat"
)

var keys []string = []string{
	"d49dcf37238dc8a7aac57dc61b9fee68f0a97f062968978b9fafa7d1033d03a9",
	"73fd6143c48e80ed3c56ea159fe7494a0b6b393a392227b422f4c3e8f1b54f98",
	"119dd32adb1daa7a4c7bf77f847fb28730785aa92947edf42fdd997b54de40dc",
	"deeda8709dea935bb772248a3144dea449ffcc13e8e5a1fd4ef20ce4e9c87837",
	"5bd208a079633befa349441bdfdc4d85ba9bd56081525008380a63ac38a407cf",
	"1d27fb4912002d58a2a42a50c97edb05c1b3dffc665dbaa42df1fe8d3d95c9b5",
	"15def52800c9d6b8ca6f3066b7767a76afc7b611786c1276165fbc61636afb68",
	"51be6ab4b2dc89f251ff2ace10f3c1cc65d6855f3e083f91f6ff8efdfd28b48c",
	"ef1ef7441bf3c6419b162f05da6037474664f198b58db7315a6f4de52414b4a0",
	"09bdf6985aabc696dc1fbeb5381aebd7a6421727343872eb2fadfc6d82486fd9",
	"15d811bf2e01f99a224cdc91d0cf76cea08e8c67905c16fee9725c9be71185c4",
	"2f83e45cf1baaea779789f755b7da72d8857aeebff19362dd9af31d3c9d14620",
	"73f04e34ac6532b19c2aae8f8e52f38df1ac8f5cd10369f92325b9b0494b0590",
	"1e2e07b69e5025537fb73770f483dc8d64f84ae3403775ef61cd36e3faf162c1",
	"8963d9bbb3911aac6d30388c786756b1c423c4fbbc95d1f96ddbddf39809e43a",
	"0422da85abc48249270b45d8de38a4cc3c02032ede1fcf0864a51092d58a2f1f",
	"8ae5c15b0e8c7cade201fdc149831aa9b11ff626a7ffd27188886cc108ad0fa8",
	"acd8f5a71d4aecfcb9ad00d32aa4bcf2a602939b6a9dd071bab443154184f805",
	"a285a922125a7481600782ad69debfbcdb0316c1e97c267aff29ef50001ec045",
	"28fd4eee78c6cd4bf78f39f8ab30c32c67c24a6223baa40e6f9c9a0e1de7cef5",
	"c5cca0c9e6f043b288c6f1aef448ab59132dab3e453671af5d0752961f013fc7",
	"46df99b051838cb6f8d1b73f232af516886bd8c4d0ee07af9a0a033c391380fd",
	"c6a06a53cbaadbb432884f36155c8f3244e244881b5ee3e92e974cfa166d793f",
	"783b90c75c63dc72e2f8d11b6f1b4de54d63825330ec76ee8db34f06b38ea211",
	"9450038f10ca2c097a8013e5121b36b422b95b04892232f930a29292d9935611",
	"e215e6246ed1cfdcf7310d4d8cdbe370f0d6a8371e4eb1089e2ae05c0e1bc10f",
	"487110939ed9d64ebbc1f300adeab358bc58875faf4ca64990fbd7fe03b78f2b",
	"824a70ea76ac81366da1d4f4ac39de851c8ac49dca456bb3f0a186ceefa269a5",
	"ba8f34fa40945560d1006a328fe70c42e35cc3d1017e72d26864cd0d1b150f15",
	"30a5dfcfd144997f428901ea88a43c8d176b19c79dde54cc58eea001aa3d246c",
	"de59f7183aca39aa245ce66a05245fecfc7e2c75884184b52b27734a4a58efa2",
	"92629e2ff5f0cb4f5f08fffe0f64492024d36f045b901efb271674b801095c5a",
	"7184c1701569e3a4c4d2ddce691edd983b81e42e09196d332e1ae2f1e062cff4",
}

const NumNodes = 16 // must not exceed the number of keys (32)

type TestData struct {
	counter [NumNodes]int
	mutex   sync.RWMutex
}

type TestNode struct {
	shh     *Whisper
	id      *ecdsa.PrivateKey
	server  *p2p.Server
	filerId uint32
}

var result TestData
var nodes [NumNodes]*TestNode
var sharedKey []byte = []byte("some arbitrary data here")
var sharedTopic TopicType = TopicType{0xF, 0x1, 0x2, 0}
var expectedMessage []byte = []byte("per rectum ad astra")

// This test does the following:
// 1. creates a chain of whisper nodes,
// 2. installs the filters with shared (predefined) parameters,
// 3. each node sends a number of random (undecryptable) messages,
// 4. first node sends one expected (decryptable) message,
// 5. checks if each node have received and decrypted exactly one message.
func TestSimulation(t *testing.T) {
	initialize(t)

	for i := 0; i < NumNodes; i++ {
		sendMsg(t, false, i)
	}

	sendMsg(t, true, 0)
	checkPropagation(t)
	stopServers()
}

func initialize(t *testing.T) {
	//glog.SetV(6)
	//glog.SetToStderr(true)

	var err error
	ip := net.IPv4(127, 0, 0, 1)
	port0 := 30303

	for i := 0; i < NumNodes; i++ {
		var node TestNode
		node.shh = NewWhisper(nil)
		node.shh.test = true
		topics := make([]TopicType, 0)
		topics = append(topics, sharedTopic)
		f := Filter{KeySym: sharedKey, Topics: topics}
		node.filerId = node.shh.Watch(&f)
		node.id, err = crypto.HexToECDSA(keys[i])
		if err != nil {
			t.Fatalf("failed convert the key: %s.", keys[i])
		}
		port := port0 + i
		addr := fmt.Sprintf(":%d", port) // e.g. ":30303"
		name := common.MakeName("whisper-go", "2.0")
		var peers []*discover.Node
		if i > 0 {
			peerNodeId := nodes[i-1].id
			peerPort := uint16(port - 1)
			peerNode := discover.PubkeyID(&peerNodeId.PublicKey)
			peer := discover.NewNode(peerNode, ip, peerPort, peerPort)
			peers = append(peers, peer)
		}

		node.server = &p2p.Server{
			Config: p2p.Config{
				PrivateKey:     node.id,
				MaxPeers:       NumNodes/2 + 1,
				Name:           name,
				Protocols:      node.shh.Protocols(),
				ListenAddr:     addr,
				NAT:            nat.Any(),
				BootstrapNodes: peers,
				StaticNodes:    peers,
				TrustedNodes:   peers,
			},
		}

		err = node.server.Start()
		if err != nil {
			t.Fatalf("failed to start server %d.", i)
		}

		nodes[i] = &node
	}
}

func stopServers() {
	for i := 0; i < NumNodes; i++ {
		n := nodes[i]
		if n != nil {
			n.shh.Unwatch(n.filerId)
			n.server.Stop()
		}
	}
}

func checkPropagation(t *testing.T) {
	if t.Failed() {
		return
	}

	const cycle = 100
	const iterations = 100

	for j := 0; j < iterations; j++ {
		time.Sleep(cycle * time.Millisecond)

		for i := 0; i < NumNodes; i++ {
			f := nodes[i].shh.GetFilter(nodes[i].filerId)
			if f == nil {
				t.Fatalf("failed to get filterId %d from node %d.", nodes[i].filerId, i)
			}

			mail := f.Retrieve()
			if !validateMail(t, i, mail) {
				return
			}

			if isTestComplete() {
				return
			}
		}
	}

	t.Fatalf("Test was not complete: timeout %d seconds.", iterations*cycle/1000)
}

func validateMail(t *testing.T, index int, mail []*ReceivedMessage) bool {
	var cnt int
	for _, m := range mail {
		if bytes.Compare(m.Payload, expectedMessage) == 0 {
			cnt++
		}
	}

	if cnt == 0 {
		// no messages received yet: nothing is wrong
		return true
	}
	if cnt > 1 {
		t.Fatalf("node %d received %d.", index, cnt)
		return false
	}

	if cnt > 0 {
		result.mutex.Lock()
		defer result.mutex.Unlock()
		result.counter[index] += cnt
		if result.counter[index] > 1 {
			t.Fatalf("node %d accumulated %d.", index, result.counter[index])
		}
	}
	return true
}

func isTestComplete() bool {
	result.mutex.RLock()
	defer result.mutex.RUnlock()

	for i := 0; i < NumNodes; i++ {
		if result.counter[i] < 1 {
			return false
		}
	}

	for i := 0; i < NumNodes; i++ {
		envelopes := nodes[i].shh.Envelopes()
		if len(envelopes) < 2 {
			return false
		}
	}

	return true
}

func sendMsg(t *testing.T, expected bool, id int) {
	if t.Failed() {
		return
	}

	opt := MessageParams{KeySym: sharedKey, Topic: sharedTopic, Payload: expectedMessage, PoW: 0.00000001}
	if !expected {
		opt.KeySym[0]++
		opt.Topic[0]++
		opt.Payload = opt.Payload[1:]
	}

	msg := NewSentMessage(&opt)
	envelope, err := msg.Wrap(&opt)
	if err != nil {
		t.Fatalf("failed to seal message.")
	}

	err = nodes[id].shh.Send(envelope)
	if err != nil {
		t.Fatalf("failed to send message.")
	}
}

func TestPeerBasic(t *testing.T) {
	InitSingleTest()

	params, err := generateMessageParams()
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d.", seed)
	}

	params.PoW = 0.001
	msg := NewSentMessage(params)
	env, err := msg.Wrap(params)
	if err != nil {
		t.Fatalf("failed Wrap with seed %d.", seed)
	}

	p := newPeer(nil, nil, nil)
	p.mark(env)
	if !p.marked(env) {
		t.Fatalf("failed mark with seed %d.", seed)
	}
}
