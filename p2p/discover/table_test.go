// Copyright 2015 The go-ethereum Authors
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

package discover

import (
	"crypto/ecdsa"
	"fmt"
	"math/rand"

	"net"
	"reflect"
	"testing"
	"testing/quick"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestTable_pingReplace(t *testing.T) {
	doit := func(newNodeIsResponding, lastInBucketIsResponding bool) {
		transport := newPingRecorder()
		tab, _ := newTable(transport, NodeID{}, &net.UDPAddr{}, "")
		defer tab.Close()
		pingSender := NewNode(MustHexID("a502af0f59b2aab7746995408c79e9ca312d2793cc997e44fc55eda62f0150bbb8c59a6f9269ba3a081518b62699ee807c7c19c20125ddfccca872608af9e370"), net.IP{}, 99, 99)

		// fill up the sender's bucket.
		last := fillBucket(tab, 253)

		// this call to bond should replace the last node
		// in its bucket if the node is not responding.
		transport.responding[last.ID] = lastInBucketIsResponding
		transport.responding[pingSender.ID] = newNodeIsResponding
		tab.bond(true, pingSender.ID, &net.UDPAddr{}, 0)

		// first ping goes to sender (bonding pingback)
		if !transport.pinged[pingSender.ID] {
			t.Error("table did not ping back sender")
		}
		if newNodeIsResponding {
			// second ping goes to oldest node in bucket
			// to see whether it is still alive.
			if !transport.pinged[last.ID] {
				t.Error("table did not ping last node in bucket")
			}
		}

		tab.mutex.Lock()
		defer tab.mutex.Unlock()
		if l := len(tab.buckets[253].entries); l != bucketSize {
			t.Errorf("wrong bucket size after bond: got %d, want %d", l, bucketSize)
		}

		if lastInBucketIsResponding || !newNodeIsResponding {
			if !contains(tab.buckets[253].entries, last.ID) {
				t.Error("last entry was removed")
			}
			if contains(tab.buckets[253].entries, pingSender.ID) {
				t.Error("new entry was added")
			}
		} else {
			if contains(tab.buckets[253].entries, last.ID) {
				t.Error("last entry was not removed")
			}
			if !contains(tab.buckets[253].entries, pingSender.ID) {
				t.Error("new entry was not added")
			}
		}
	}

	doit(true, true)
	doit(false, true)
	doit(true, false)
	doit(false, false)
}

func TestBucket_bumpNoDuplicates(t *testing.T) {
	t.Parallel()
	cfg := &quick.Config{
		MaxCount: 1000,
		Rand:     rand.New(rand.NewSource(time.Now().Unix())),
		Values: func(args []reflect.Value, rand *rand.Rand) {
			// generate a random list of nodes. this will be the content of the bucket.
			n := rand.Intn(bucketSize-1) + 1
			nodes := make([]*Node, n)
			for i := range nodes {
				nodes[i] = nodeAtDistance(common.Hash{}, 200)
			}
			args[0] = reflect.ValueOf(nodes)
			// generate random bump positions.
			bumps := make([]int, rand.Intn(100))
			for i := range bumps {
				bumps[i] = rand.Intn(len(nodes))
			}
			args[1] = reflect.ValueOf(bumps)
		},
	}

	prop := func(nodes []*Node, bumps []int) (ok bool) {
		b := &bucket{entries: make([]*Node, len(nodes))}
		copy(b.entries, nodes)
		for i, pos := range bumps {
			b.bump(b.entries[pos])
			if hasDuplicates(b.entries) {
				t.Logf("bucket has duplicates after %d/%d bumps:", i+1, len(bumps))
				for _, n := range b.entries {
					t.Logf("  %p", n)
				}
				return false
			}
		}
		return true
	}
	if err := quick.Check(prop, cfg); err != nil {
		t.Error(err)
	}
}

// fillBucket inserts nodes into the given bucket until
// it is full. The node's IDs dont correspond to their
// hashes.
func fillBucket(tab *Table, ld int) (last *Node) {
	b := tab.buckets[ld]
	for len(b.entries) < bucketSize {
		b.entries = append(b.entries, nodeAtDistance(tab.self.sha, ld))
	}
	return b.entries[bucketSize-1]
}

// nodeAtDistance creates a node for which logdist(base, n.sha) == ld.
// The node's ID does not correspond to n.sha.
func nodeAtDistance(base common.Hash, ld int) (n *Node) {
	n = new(Node)
	n.sha = hashAtDistance(base, ld)
	n.IP = net.IP{10, 0, 2, byte(ld)}
	copy(n.ID[:], n.sha[:]) // ensure the node still has a unique ID
	return n
}

type pingRecorder struct{ responding, pinged map[NodeID]bool }

func newPingRecorder() *pingRecorder {
	return &pingRecorder{make(map[NodeID]bool), make(map[NodeID]bool)}
}

func (t *pingRecorder) findnode(toid NodeID, toaddr *net.UDPAddr, target NodeID) ([]*Node, error) {
	panic("findnode called on pingRecorder")
}
func (t *pingRecorder) close() {}
func (t *pingRecorder) waitping(from NodeID) error {
	return nil // remote always pings
}
func (t *pingRecorder) ping(toid NodeID, toaddr *net.UDPAddr) error {
	t.pinged[toid] = true
	if t.responding[toid] {
		return nil
	} else {
		return errTimeout
	}
}

func TestTable_closest(t *testing.T) {
	t.Parallel()

	test := func(test *closeTest) bool {
		// for any node table, Target and N
		tab, _ := newTable(nil, test.Self, &net.UDPAddr{}, "")
		defer tab.Close()
		tab.stuff(test.All)

		// check that doClosest(Target, N) returns nodes
		result := tab.closest(test.Target, test.N).entries
		if hasDuplicates(result) {
			t.Errorf("result contains duplicates")
			return false
		}
		if !sortedByDistanceTo(test.Target, result) {
			t.Errorf("result is not sorted by distance to target")
			return false
		}

		// check that the number of results is min(N, tablen)
		wantN := test.N
		if tlen := tab.len(); tlen < test.N {
			wantN = tlen
		}
		if len(result) != wantN {
			t.Errorf("wrong number of nodes: got %d, want %d", len(result), wantN)
			return false
		} else if len(result) == 0 {
			return true // no need to check distance
		}

		// check that the result nodes have minimum distance to target.
		for _, b := range tab.buckets {
			for _, n := range b.entries {
				if contains(result, n.ID) {
					continue // don't run the check below for nodes in result
				}
				farthestResult := result[len(result)-1].sha
				if distcmp(test.Target, n.sha, farthestResult) < 0 {
					t.Errorf("table contains node that is closer to target but it's not in result")
					t.Logf("  Target:          %v", test.Target)
					t.Logf("  Farthest Result: %v", farthestResult)
					t.Logf("  ID:              %v", n.ID)
					return false
				}
			}
		}
		return true
	}
	if err := quick.Check(test, quickcfg()); err != nil {
		t.Error(err)
	}
}

func TestTable_ReadRandomNodesGetAll(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 200,
		Rand:     rand.New(rand.NewSource(time.Now().Unix())),
		Values: func(args []reflect.Value, rand *rand.Rand) {
			args[0] = reflect.ValueOf(make([]*Node, rand.Intn(1000)))
		},
	}
	test := func(buf []*Node) bool {
		tab, _ := newTable(nil, NodeID{}, &net.UDPAddr{}, "")
		defer tab.Close()
		for i := 0; i < len(buf); i++ {
			ld := cfg.Rand.Intn(len(tab.buckets))
			tab.stuff([]*Node{nodeAtDistance(tab.self.sha, ld)})
		}
		gotN := tab.ReadRandomNodes(buf)
		if gotN != tab.len() {
			t.Errorf("wrong number of nodes, got %d, want %d", gotN, tab.len())
			return false
		}
		if hasDuplicates(buf[:gotN]) {
			t.Errorf("result contains duplicates")
			return false
		}
		return true
	}
	if err := quick.Check(test, cfg); err != nil {
		t.Error(err)
	}
}

type closeTest struct {
	Self   NodeID
	Target common.Hash
	All    []*Node
	N      int
}

func (*closeTest) Generate(rand *rand.Rand, size int) reflect.Value {
	t := &closeTest{
		Self:   gen(NodeID{}, rand).(NodeID),
		Target: gen(common.Hash{}, rand).(common.Hash),
		N:      rand.Intn(bucketSize),
	}
	for _, id := range gen([]NodeID{}, rand).([]NodeID) {
		t.All = append(t.All, &Node{ID: id})
	}
	return reflect.ValueOf(t)
}

func TestTable_Lookup(t *testing.T) {
	self := nodeAtDistance(common.Hash{}, 0)
	tab, _ := newTable(lookupTestnet, self.ID, &net.UDPAddr{}, "")
	defer tab.Close()

	// lookup on empty table returns no nodes
	if results := tab.Lookup(lookupTestnet.target); len(results) > 0 {
		t.Fatalf("lookup on empty table returned %d results: %#v", len(results), results)
	}
	// seed table with initial node (otherwise lookup will terminate immediately)
	seed := NewNode(lookupTestnet.dists[256][0], net.IP{}, 256, 0)
	tab.stuff([]*Node{seed})

	results := tab.Lookup(lookupTestnet.target)
	t.Logf("results:")
	for _, e := range results {
		t.Logf("  ld=%d, %x", logdist(lookupTestnet.targetSha, e.sha), e.sha[:])
	}
	if len(results) != bucketSize {
		t.Errorf("wrong number of results: got %d, want %d", len(results), bucketSize)
	}
	if hasDuplicates(results) {
		t.Errorf("result set contains duplicate entries")
	}
	if !sortedByDistanceTo(lookupTestnet.targetSha, results) {
		t.Errorf("result set not sorted by distance to target")
	}
	// TODO: check result nodes are actually closest
}

// This is the test network for the Lookup test.
// The nodes were obtained by running testnet.mine with a random NodeID as target.
var lookupTestnet = &preminedTestnet{
	target:    MustHexID("166aea4f556532c6d34e8b740e5d314af7e9ac0ca79833bd751d6b665f12dfd38ec563c363b32f02aef4a80b44fd3def94612d497b99cb5f17fd24de454927ec"),
	targetSha: common.Hash{0x5c, 0x94, 0x4e, 0xe5, 0x1c, 0x5a, 0xe9, 0xf7, 0x2a, 0x95, 0xec, 0xcb, 0x8a, 0xed, 0x3, 0x74, 0xee, 0xcb, 0x51, 0x19, 0xd7, 0x20, 0xcb, 0xea, 0x68, 0x13, 0xe8, 0xe0, 0xd6, 0xad, 0x92, 0x61},
	dists: [257][]NodeID{
		240: []NodeID{
			MustHexID("2001ad5e3e80c71b952161bc0186731cf5ffe942d24a79230a0555802296238e57ea7a32f5b6f18564eadc1c65389448481f8c9338df0a3dbd18f708cbc2cbcb"),
			MustHexID("6ba3f4f57d084b6bf94cc4555b8c657e4a8ac7b7baf23c6874efc21dd1e4f56b7eb2721e07f5242d2f1d8381fc8cae535e860197c69236798ba1ad231b105794"),
		},
		244: []NodeID{
			MustHexID("696ba1f0a9d55c59246f776600542a9e6432490f0cd78f8bb55a196918df2081a9b521c3c3ba48e465a75c10768807717f8f689b0b4adce00e1c75737552a178"),
		},
		246: []NodeID{
			MustHexID("d6d32178bdc38416f46ffb8b3ec9e4cb2cfff8d04dd7e4311a70e403cb62b10be1b447311b60b4f9ee221a8131fc2cbd45b96dd80deba68a949d467241facfa8"),
			MustHexID("3ea3d04a43a3dfb5ac11cffc2319248cf41b6279659393c2f55b8a0a5fc9d12581a9d97ef5d8ff9b5abf3321a290e8f63a4f785f450dc8a672aba3ba2ff4fdab"),
			MustHexID("2fc897f05ae585553e5c014effd3078f84f37f9333afacffb109f00ca8e7a3373de810a3946be971cbccdfd40249f9fe7f322118ea459ac71acca85a1ef8b7f4"),
		},
		247: []NodeID{
			MustHexID("3155e1427f85f10a5c9a7755877748041af1bcd8d474ec065eb33df57a97babf54bfd2103575fa829115d224c523596b401065a97f74010610fce76382c0bf32"),
			MustHexID("312c55512422cf9b8a4097e9a6ad79402e87a15ae909a4bfefa22398f03d20951933beea1e4dfa6f968212385e829f04c2d314fc2d4e255e0d3bc08792b069db"),
			MustHexID("38643200b172dcfef857492156971f0e6aa2c538d8b74010f8e140811d53b98c765dd2d96126051913f44582e8c199ad7c6d6819e9a56483f637feaac9448aac"),
			MustHexID("8dcab8618c3253b558d459da53bd8fa68935a719aff8b811197101a4b2b47dd2d47295286fc00cc081bb542d760717d1bdd6bec2c37cd72eca367d6dd3b9df73"),
			MustHexID("8b58c6073dd98bbad4e310b97186c8f822d3a5c7d57af40e2136e88e315afd115edb27d2d0685a908cfe5aa49d0debdda6e6e63972691d6bd8c5af2d771dd2a9"),
			MustHexID("2cbb718b7dc682da19652e7d9eb4fefaf7b7147d82c1c2b6805edf77b85e29fde9f6da195741467ff2638dc62c8d3e014ea5686693c15ed0080b6de90354c137"),
			MustHexID("e84027696d3f12f2de30a9311afea8fbd313c2360daff52bb5fc8c7094d5295758bec3134e4eef24e4cdf377b40da344993284628a7a346eba94f74160998feb"),
			MustHexID("f1357a4f04f9d33753a57c0b65ba20a5d8777abbffd04e906014491c9103fb08590e45548d37aa4bd70965e2e81ddba94f31860348df01469eec8c1829200a68"),
			MustHexID("4ab0a75941b12892369b4490a1928c8ca52a9ad6d3dffbd1d8c0b907bc200fe74c022d011ec39b64808a39c0ca41f1d3254386c3e7733e7044c44259486461b6"),
			MustHexID("d45150a72dc74388773e68e03133a3b5f51447fe91837d566706b3c035ee4b56f160c878c6273394daee7f56cc398985269052f22f75a8057df2fe6172765354"),
		},
		248: []NodeID{
			MustHexID("6aadfce366a189bab08ac84721567483202c86590642ea6d6a14f37ca78d82bdb6509eb7b8b2f6f63c78ae3ae1d8837c89509e41497d719b23ad53dd81574afa"),
			MustHexID("a605ecfd6069a4cf4cf7f5840e5bc0ce10d23a3ac59e2aaa70c6afd5637359d2519b4524f56fc2ca180cdbebe54262f720ccaae8c1b28fd553c485675831624d"),
			MustHexID("29701451cb9448ca33fc33680b44b840d815be90146eb521641efbffed0859c154e8892d3906eae9934bfacee72cd1d2fa9dd050fd18888eea49da155ab0efd2"),
			MustHexID("3ed426322dee7572b08592e1e079f8b6c6b30e10e6243edd144a6a48fdbdb83df73a6e41b1143722cb82604f2203a32758610b5d9544f44a1a7921ba001528c1"),
			MustHexID("b2e2a2b7fdd363572a3256e75435fab1da3b16f7891a8bd2015f30995dae665d7eabfd194d87d99d5df628b4bbc7b04e5b492c596422dd8272746c7a1b0b8e4f"),
			MustHexID("0c69c9756162c593e85615b814ce57a2a8ca2df6c690b9c4e4602731b61e1531a3bbe3f7114271554427ffabea80ad8f36fa95a49fa77b675ae182c6ccac1728"),
			MustHexID("8d28be21d5a97b0876442fa4f5e5387f5bf3faad0b6f13b8607b64d6e448c0991ca28dd7fe2f64eb8eadd7150bff5d5666aa6ed868b84c71311f4ba9a38569dd"),
			MustHexID("2c677e1c64b9c9df6359348a7f5f33dc79e22f0177042486d125f8b6ca7f0dc756b1f672aceee5f1746bcff80aaf6f92a8dc0c9fbeb259b3fa0da060de5ab7e8"),
			MustHexID("3994880f94a8678f0cd247a43f474a8af375d2a072128da1ad6cae84a244105ff85e94fc7d8496f639468de7ee998908a91c7e33ef7585fff92e984b210941a1"),
			MustHexID("b45a9153c08d002a48090d15d61a7c7dad8c2af85d4ff5bd36ce23a9a11e0709bf8d56614c7b193bc028c16cbf7f20dfbcc751328b64a924995d47b41e452422"),
			MustHexID("057ab3a9e53c7a84b0f3fc586117a525cdd18e313f52a67bf31798d48078e325abe5cfee3f6c2533230cb37d0549289d692a29dd400e899b8552d4b928f6f907"),
			MustHexID("0ddf663d308791eb92e6bd88a2f8cb45e4f4f35bb16708a0e6ff7f1362aa6a73fedd0a1b1557fb3365e38e1b79d6918e2fae2788728b70c9ab6b51a3b94a4338"),
			MustHexID("f637e07ff50cc1e3731735841c4798411059f2023abcf3885674f3e8032531b0edca50fd715df6feb489b6177c345374d64f4b07d257a7745de393a107b013a5"),
			MustHexID("e24ec7c6eec094f63c7b3239f56d311ec5a3e45bc4e622a1095a65b95eea6fe13e29f3b6b7a2cbfe40906e3989f17ac834c3102dd0cadaaa26e16ee06d782b72"),
			MustHexID("b76ea1a6fd6506ef6e3506a4f1f60ed6287fff8114af6141b2ff13e61242331b54082b023cfea5b3083354a4fb3f9eb8be01fb4a518f579e731a5d0707291a6b"),
			MustHexID("9b53a37950ca8890ee349b325032d7b672cab7eced178d3060137b24ef6b92a43977922d5bdfb4a3409a2d80128e02f795f9dae6d7d99973ad0e23a2afb8442f"),
		},
		249: []NodeID{
			MustHexID("675ae65567c3c72c50c73bc0fd4f61f202ea5f93346ca57b551de3411ccc614fad61cb9035493af47615311b9d44ee7a161972ee4d77c28fe1ec029d01434e6a"),
			MustHexID("8eb81408389da88536ae5800392b16ef5109d7ea132c18e9a82928047ecdb502693f6e4a4cdd18b54296caf561db937185731456c456c98bfe7de0baf0eaa495"),
			MustHexID("2adba8b1612a541771cb93a726a38a4b88e97b18eced2593eb7daf82f05a5321ca94a72cc780c306ff21e551a932fc2c6d791e4681907b5ceab7f084c3fa2944"),
			MustHexID("b1b4bfbda514d9b8f35b1c28961da5d5216fe50548f4066f69af3b7666a3b2e06eac646735e963e5c8f8138a2fb95af15b13b23ff00c6986eccc0efaa8ee6fb4"),
			MustHexID("d2139281b289ad0e4d7b4243c4364f5c51aac8b60f4806135de06b12b5b369c9e43a6eb494eab860d115c15c6fbb8c5a1b0e382972e0e460af395b8385363de7"),
			MustHexID("4a693df4b8fc5bdc7cec342c3ed2e228d7c5b4ab7321ddaa6cccbeb45b05a9f1d95766b4002e6d4791c2deacb8a667aadea6a700da28a3eea810a30395701bbc"),
			MustHexID("ab41611195ec3c62bb8cd762ee19fb182d194fd141f4a66780efbef4b07ce916246c022b841237a3a6b512a93431157edd221e854ed2a259b72e9c5351f44d0c"),
			MustHexID("68e8e26099030d10c3c703ae7045c0a48061fb88058d853b3e67880014c449d4311014da99d617d3150a20f1a3da5e34bf0f14f1c51fe4dd9d58afd222823176"),
			MustHexID("3fbcacf546fb129cd70fc48de3b593ba99d3c473798bc309292aca280320e0eacc04442c914cad5c4cf6950345ba79b0d51302df88285d4e83ee3fe41339eee7"),
			MustHexID("1d4a623659f7c8f80b6c3939596afdf42e78f892f682c768ad36eb7bfba402dbf97aea3a268f3badd8fe7636be216edf3d67ee1e08789ebbc7be625056bd7109"),
			MustHexID("a283c474ab09da02bbc96b16317241d0627646fcc427d1fe790b76a7bf1989ced90f92101a973047ae9940c92720dffbac8eff21df8cae468a50f72f9e159417"),
			MustHexID("dbf7e5ad7f87c3dfecae65d87c3039e14ed0bdc56caf00ce81931073e2e16719d746295512ff7937a15c3b03603e7c41a4f9df94fcd37bb200dd8f332767e9cb"),
			MustHexID("caaa070a26692f64fc77f30d7b5ae980d419b4393a0f442b1c821ef58c0862898b0d22f74a4f8c5d83069493e3ec0b92f17dc1fe6e4cd437c1ec25039e7ce839"),
			MustHexID("874cc8d1213beb65c4e0e1de38ef5d8165235893ac74ab5ea937c885eaab25c8d79dad0456e9fd3e9450626cac7e107b004478fb59842f067857f39a47cee695"),
			MustHexID("d94193f236105010972f5df1b7818b55846592a0445b9cdc4eaed811b8c4c0f7c27dc8cc9837a4774656d6b34682d6d329d42b6ebb55da1d475c2474dc3dfdf4"),
			MustHexID("edd9af6aded4094e9785637c28fccbd3980cbe28e2eb9a411048a23c2ace4bd6b0b7088a7817997b49a3dd05fc6929ca6c7abbb69438dbdabe65e971d2a794b2"),
		},
		250: []NodeID{
			MustHexID("53a5bd1215d4ab709ae8fdc2ced50bba320bced78bd9c5dc92947fb402250c914891786db0978c898c058493f86fc68b1c5de8a5cb36336150ac7a88655b6c39"),
			MustHexID("b7f79e3ab59f79262623c9ccefc8f01d682323aee56ffbe295437487e9d5acaf556a9c92e1f1c6a9601f2b9eb6b027ae1aeaebac71d61b9b78e88676efd3e1a3"),
			MustHexID("d374bf7e8d7ffff69cc00bebff38ef5bc1dcb0a8d51c1a3d70e61ac6b2e2d6617109254b0ac224354dfbf79009fe4239e09020c483cc60c071e00b9238684f30"),
			MustHexID("1e1eac1c9add703eb252eb991594f8f5a173255d526a855fab24ae57dc277e055bc3c7a7ae0b45d437c4f47a72d97eb7b126f2ba344ba6c0e14b2c6f27d4b1e6"),
			MustHexID("ae28953f63d4bc4e706712a59319c111f5ff8f312584f65d7436b4cd3d14b217b958f8486bad666b4481fe879019fb1f767cf15b3e3e2711efc33b56d460448a"),
			MustHexID("934bb1edf9c7a318b82306aca67feb3d6b434421fa275d694f0b4927afd8b1d3935b727fd4ff6e3d012e0c82f1824385174e8c6450ade59c2a43281a4b3446b6"),
			MustHexID("9eef3f28f70ce19637519a0916555bf76d26de31312ac656cf9d3e379899ea44e4dd7ffcce923b4f3563f8a00489a34bd6936db0cbb4c959d32c49f017e07d05"),
			MustHexID("82200872e8f871c48f1fad13daec6478298099b591bb3dbc4ef6890aa28ebee5860d07d70be62f4c0af85085a90ae8179ee8f937cf37915c67ea73e704b03ee7"),
			MustHexID("6c75a5834a08476b7fc37ff3dc2011dc3ea3b36524bad7a6d319b18878fad813c0ba76d1f4555cacd3890c865438c21f0e0aed1f80e0a157e642124c69f43a11"),
			MustHexID("995b873742206cb02b736e73a88580c2aacb0bd4a3c97a647b647bcab3f5e03c0e0736520a8b3600da09edf4248991fb01091ec7ff3ec7cdc8a1beae011e7aae"),
			MustHexID("c773a056594b5cdef2e850d30891ff0e927c3b1b9c35cd8e8d53a1017001e237468e1ece3ae33d612ca3e6abb0a9169aa352e9dcda358e5af2ad982b577447db"),
			MustHexID("2b46a5f6923f475c6be99ec6d134437a6d11f6bb4b4ac6bcd94572fa1092639d1c08aeefcb51f0912f0a060f71d4f38ee4da70ecc16010b05dd4a674aab14c3a"),
			MustHexID("af6ab501366debbaa0d22e20e9688f32ef6b3b644440580fd78de4fe0e99e2a16eb5636bbae0d1c259df8ddda77b35b9a35cbc36137473e9c68fbc9d203ba842"),
			MustHexID("c9f6f2dd1a941926f03f770695bda289859e85fabaf94baaae20b93e5015dc014ba41150176a36a1884adb52f405194693e63b0c464a6891cc9cc1c80d450326"),
			MustHexID("5b116f0751526868a909b61a30b0c5282c37df6925cc03ddea556ef0d0602a9595fd6c14d371f8ed7d45d89918a032dcd22be4342a8793d88fdbeb3ca3d75bd7"),
			MustHexID("50f3222fb6b82481c7c813b2172e1daea43e2710a443b9c2a57a12bd160dd37e20f87aa968c82ad639af6972185609d47036c0d93b4b7269b74ebd7073221c10"),
		},
		251: []NodeID{
			MustHexID("9b8f702a62d1bee67bedfeb102eca7f37fa1713e310f0d6651cc0c33ea7c5477575289ccd463e5a2574a00a676a1fdce05658ba447bb9d2827f0ba47b947e894"),
			MustHexID("b97532eb83054ed054b4abdf413bb30c00e4205545c93521554dbe77faa3cfaa5bd31ef466a107b0b34a71ec97214c0c83919720142cddac93aa7a3e928d4708"),
			MustHexID("2f7a5e952bfb67f2f90b8441b5fadc9ee13b1dcde3afeeb3dd64bf937f86663cc5c55d1fa83952b5422763c7df1b7f2794b751c6be316ebc0beb4942e65ab8c1"),
			MustHexID("42c7483781727051a0b3660f14faf39e0d33de5e643702ae933837d036508ab856ce7eec8ec89c4929a4901256e5233a3d847d5d4893f91bcf21835a9a880fee"),
			MustHexID("873bae27bf1dc854408fba94046a53ab0c965cebe1e4e12290806fc62b88deb1f4a47f9e18f78fc0e7913a0c6e42ac4d0fc3a20cea6bc65f0c8a0ca90b67521e"),
			MustHexID("a7e3a370bbd761d413f8d209e85886f68bf73d5c3089b2dc6fa42aab1ecb5162635497eed95dee2417f3c9c74a3e76319625c48ead2e963c7de877cd4551f347"),
			MustHexID("528597534776a40df2addaaea15b6ff832ce36b9748a265768368f657e76d58569d9f30dbb91e91cf0ae7efe8f402f17aa0ae15f5c55051ba03ba830287f4c42"),
			MustHexID("461d8bd4f13c3c09031fdb84f104ed737a52f630261463ce0bdb5704259bab4b737dda688285b8444dbecaecad7f50f835190b38684ced5e90c54219e5adf1bc"),
			MustHexID("6ec50c0be3fd232737090fc0111caaf0bb6b18f72be453428087a11a97fd6b52db0344acbf789a689bd4f5f50f79017ea784f8fd6fe723ad6ae675b9e3b13e21"),
			MustHexID("12fc5e2f77a83fdcc727b79d8ae7fe6a516881138d3011847ee136b400fed7cfba1f53fd7a9730253c7aa4f39abeacd04f138417ba7fcb0f36cccc3514e0dab6"),
			MustHexID("4fdbe75914ccd0bce02101606a1ccf3657ec963e3b3c20239d5fec87673fe446d649b4f15f1fe1a40e6cfbd446dda2d31d40bb602b1093b8fcd5f139ba0eb46a"),
			MustHexID("3753668a0f6281e425ea69b52cb2d17ab97afbe6eb84cf5d25425bc5e53009388857640668fadd7c110721e6047c9697803bd8a6487b43bb343bfa32ebf24039"),
			MustHexID("2e81b16346637dec4410fd88e527346145b9c0a849dbf2628049ac7dae016c8f4305649d5659ec77f1e8a0fac0db457b6080547226f06283598e3740ad94849a"),
			MustHexID("802c3cc27f91c89213223d758f8d2ecd41135b357b6d698f24d811cdf113033a81c38e0bdff574a5c005b00a8c193dc2531f8c1fa05fa60acf0ab6f2858af09f"),
			MustHexID("fcc9a2e1ac3667026ff16192876d1813bb75abdbf39b929a92863012fe8b1d890badea7a0de36274d5c1eb1e8f975785532c50d80fd44b1a4b692f437303393f"),
			MustHexID("6d8b3efb461151dd4f6de809b62726f5b89e9b38e9ba1391967f61cde844f7528fecf821b74049207cee5a527096b31f3ad623928cd3ce51d926fa345a6b2951"),
		},
		252: []NodeID{
			MustHexID("f1ae93157cc48c2075dd5868fbf523e79e06caf4b8198f352f6e526680b78ff4227263de92612f7d63472bd09367bb92a636fff16fe46ccf41614f7a72495c2a"),
			MustHexID("587f482d111b239c27c0cb89b51dd5d574db8efd8de14a2e6a1400c54d4567e77c65f89c1da52841212080b91604104768350276b6682f2f961cdaf4039581c7"),
			MustHexID("e3f88274d35cefdaabdf205afe0e80e936cc982b8e3e47a84ce664c413b29016a4fb4f3a3ebae0a2f79671f8323661ed462bf4390af94c424dc8ace0c301b90f"),
			MustHexID("0ddc736077da9a12ba410dc5ea63cbcbe7659dd08596485b2bff3435221f82c10d263efd9af938e128464be64a178b7cd22e19f400d5802f4c9df54bf89f2619"),
			MustHexID("784aa34d833c6ce63fcc1279630113c3272e82c4ae8c126c5a52a88ac461b6baeed4244e607b05dc14e5b2f41c70a273c3804dea237f14f7a1e546f6d1309d14"),
			MustHexID("f253a2c354ee0e27cfcae786d726753d4ad24be6516b279a936195a487de4a59dbc296accf20463749ff55293263ed8c1b6365eecb248d44e75e9741c0d18205"),
			MustHexID("a1910b80357b3ad9b4593e0628922939614dc9056a5fbf477279c8b2c1d0b4b31d89a0c09d0d41f795271d14d3360ef08a3f821e65e7e1f56c07a36afe49c7c5"),
			MustHexID("f1168552c2efe541160f0909b0b4a9d6aeedcf595cdf0e9b165c97e3e197471a1ee6320e93389edfba28af6eaf10de98597ad56e7ab1b504ed762451996c3b98"),
			MustHexID("b0c8e5d2c8634a7930e1a6fd082e448c6cf9d2d8b7293558b59238815a4df926c286bf297d2049f14e8296a6eb3256af614ec1812c4f2bbe807673b58bf14c8c"),
			MustHexID("0fb346076396a38badc342df3679b55bd7f40a609ab103411fe45082c01f12ea016729e95914b2b5540e987ff5c9b133e85862648e7f36abdfd23100d248d234"),
			MustHexID("f736e0cc83417feaa280d9483f5d4d72d1b036cd0c6d9cbdeb8ac35ceb2604780de46dddaa32a378474e1d5ccdf79b373331c30c7911ade2ae32f98832e5de1f"),
			MustHexID("8b02991457602f42b38b342d3f2259ae4100c354b3843885f7e4e07bd644f64dab94bb7f38a3915f8b7f11d8e3f81c28e07a0078cf79d7397e38a7b7e0c857e2"),
			MustHexID("9221d9f04a8a184993d12baa91116692bb685f887671302999d69300ad103eb2d2c75a09d8979404c6dd28f12362f58a1a43619c493d9108fd47588a23ce5824"),
			MustHexID("652797801744dada833fff207d67484742eea6835d695925f3e618d71b68ec3c65bdd85b4302b2cdcb835ad3f94fd00d8da07e570b41bc0d2bcf69a8de1b3284"),
			MustHexID("d84f06fe64debc4cd0625e36d19b99014b6218375262cc2209202bdbafd7dffcc4e34ce6398e182e02fd8faeed622c3e175545864902dfd3d1ac57647cddf4c6"),
			MustHexID("d0ed87b294f38f1d741eb601020eeec30ac16331d05880fe27868f1e454446de367d7457b41c79e202eaf9525b029e4f1d7e17d85a55f83a557c005c68d7328a"),
		},
		253: []NodeID{
			MustHexID("ad4485e386e3cc7c7310366a7c38fb810b8896c0d52e55944bfd320ca294e7912d6c53c0a0cf85e7ce226e92491d60430e86f8f15cda0161ed71893fb4a9e3a1"),
			MustHexID("36d0e7e5b7734f98c6183eeeb8ac5130a85e910a925311a19c4941b1290f945d4fc3996b12ef4966960b6fa0fb29b1604f83a0f81bd5fd6398d2e1a22e46af0c"),
			MustHexID("7d307d8acb4a561afa23bdf0bd945d35c90245e26345ec3a1f9f7df354222a7cdcb81339c9ed6744526c27a1a0c8d10857e98df942fa433602facac71ac68a31"),
			MustHexID("d97bf55f88c83fae36232661af115d66ca600fc4bd6d1fb35ff9bb4dad674c02cf8c8d05f317525b5522250db58bb1ecafb7157392bf5aa61b178c61f098d995"),
			MustHexID("7045d678f1f9eb7a4613764d17bd5698796494d0bf977b16f2dbc272b8a0f7858a60805c022fc3d1fe4f31c37e63cdaca0416c0d053ef48a815f8b19121605e0"),
			MustHexID("14e1f21418d445748de2a95cd9a8c3b15b506f86a0acabd8af44bb968ce39885b19c8822af61b3dd58a34d1f265baec30e3ae56149dc7d2aa4a538f7319f69c8"),
			MustHexID("b9453d78281b66a4eac95a1546017111eaaa5f92a65d0de10b1122940e92b319728a24edf4dec6acc412321b1c95266d39c7b3a5d265c629c3e49a65fb022c09"),
			MustHexID("e8a49248419e3824a00d86af422f22f7366e2d4922b304b7169937616a01d9d6fa5abf5cc01061a352dc866f48e1fa2240dbb453d872b1d7be62bdfc1d5e248c"),
			MustHexID("bebcff24b52362f30e0589ee573ce2d86f073d58d18e6852a592fa86ceb1a6c9b96d7fb9ec7ed1ed98a51b6743039e780279f6bb49d0a04327ac7a182d9a56f6"),
			MustHexID("d0835e5a4291db249b8d2fca9f503049988180c7d247bedaa2cf3a1bad0a76709360a85d4f9a1423b2cbc82bb4d94b47c0cde20afc430224834c49fe312a9ae3"),
			MustHexID("6b087fe2a2da5e4f0b0f4777598a4a7fb66bf77dbd5bfc44e8a7eaa432ab585a6e226891f56a7d4f5ed11a7c57b90f1661bba1059590ca4267a35801c2802913"),
			MustHexID("d901e5bde52d1a0f4ddf010a686a53974cdae4ebe5c6551b3c37d6b6d635d38d5b0e5f80bc0186a2c7809dbf3a42870dd09643e68d32db896c6da8ba734579e7"),
			MustHexID("96419fb80efae4b674402bb969ebaab86c1274f29a83a311e24516d36cdf148fe21754d46c97688cdd7468f24c08b13e4727c29263393638a3b37b99ff60ebca"),
			MustHexID("7b9c1889ae916a5d5abcdfb0aaedcc9c6f9eb1c1a4f68d0c2d034fe79ac610ce917c3abc670744150fa891bfcd8ab14fed6983fca964de920aa393fa7b326748"),
			MustHexID("7a369b2b8962cc4c65900be046482fbf7c14f98a135bbbae25152c82ad168fb2097b3d1429197cf46d3ce9fdeb64808f908a489cc6019725db040060fdfe5405"),
			MustHexID("47bcae48288da5ecc7f5058dfa07cf14d89d06d6e449cb946e237aa6652ea050d9f5a24a65efdc0013ccf232bf88670979eddef249b054f63f38da9d7796dbd8"),
		},
		254: []NodeID{
			MustHexID("099739d7abc8abd38ecc7a816c521a1168a4dbd359fa7212a5123ab583ffa1cf485a5fed219575d6475dbcdd541638b2d3631a6c7fce7474e7fe3cba1d4d5853"),
			MustHexID("c2b01603b088a7182d0cf7ef29fb2b04c70acb320fccf78526bf9472e10c74ee70b3fcfa6f4b11d167bd7d3bc4d936b660f2c9bff934793d97cb21750e7c3d31"),
			MustHexID("20e4d8f45f2f863e94b45548c1ef22a11f7d36f263e4f8623761e05a64c4572379b000a52211751e2561b0f14f4fc92dd4130410c8ccc71eb4f0e95a700d4ca9"),
			MustHexID("27f4a16cc085e72d86e25c98bd2eca173eaaee7565c78ec5a52e9e12b2211f35de81b5b45e9195de2ebfe29106742c59112b951a04eb7ae48822911fc1f9389e"),
			MustHexID("55db5ee7d98e7f0b1c3b9d5be6f2bc619a1b86c3cdd513160ad4dcf267037a5fffad527ac15d50aeb32c59c13d1d4c1e567ebbf4de0d25236130c8361f9aac63"),
			MustHexID("883df308b0130fc928a8559fe50667a0fff80493bc09685d18213b2db241a3ad11310ed86b0ef662b3ce21fc3d9aa7f3fc24b8d9afe17c7407e9afd3345ae548"),
			MustHexID("c7af968cc9bc8200c3ee1a387405f7563be1dce6710a3439f42ea40657d0eae9d2b3c16c42d779605351fcdece4da637b9804e60ca08cfb89aec32c197beffa6"),
			MustHexID("3e66f2b788e3ff1d04106b80597915cd7afa06c405a7ae026556b6e583dca8e05cfbab5039bb9a1b5d06083ffe8de5780b1775550e7218f5e98624bf7af9a0a8"),
			MustHexID("4fc7f53764de3337fdaec0a711d35d3a923e72fa65025444d12230b3552ed43d9b2d1ad08ccb11f2d50c58809e6dd74dde910e195294fca3b47ae5a3967cc479"),
			MustHexID("bafdfdcf6ccaa989436752fa97c77477b6baa7deb374b16c095492c529eb133e8e2f99e1977012b64767b9d34b2cf6d2048ed489bd822b5139b523f6a423167b"),
			MustHexID("7f5d78008a4312fe059104ce80202c82b8915c2eb4411c6b812b16f7642e57c00f2c9425121f5cbac4257fe0b3e81ef5dea97ea2dbaa98f6a8b6fd4d1e5980bb"),
			MustHexID("598c37fe78f922751a052f463aeb0cb0bc7f52b7c2a4cf2da72ec0931c7c32175d4165d0f8998f7320e87324ac3311c03f9382a5385c55f0407b7a66b2acd864"),
			MustHexID("f758c4136e1c148777a7f3275a76e2db0b2b04066fd738554ec398c1c6cc9fb47e14a3b4c87bd47deaeab3ffd2110514c3855685a374794daff87b605b27ee2e"),
			MustHexID("0307bb9e4fd865a49dcf1fe4333d1b944547db650ab580af0b33e53c4fef6c789531110fac801bbcbce21fc4d6f61b6d5b24abdf5b22e3030646d579f6dca9c2"),
			MustHexID("82504b6eb49bb2c0f91a7006ce9cefdbaf6df38706198502c2e06601091fc9dc91e4f15db3410d45c6af355bc270b0f268d3dff560f956985c7332d4b10bd1ed"),
			MustHexID("b39b5b677b45944ceebe76e76d1f051de2f2a0ec7b0d650da52135743e66a9a5dba45f638258f9a7545d9a790c7fe6d3fdf82c25425c7887323e45d27d06c057"),
		},
		255: []NodeID{
			MustHexID("5c4d58d46e055dd1f093f81ee60a675e1f02f54da6206720adee4dccef9b67a31efc5c2a2949c31a04ee31beadc79aba10da31440a1f9ff2a24093c63c36d784"),
			MustHexID("ea72161ffdd4b1e124c7b93b0684805f4c4b58d617ed498b37a145c670dbc2e04976f8785583d9c805ffbf343c31d492d79f841652bbbd01b61ed85640b23495"),
			MustHexID("51caa1d93352d47a8e531692a3612adac1e8ac68d0a200d086c1c57ae1e1a91aa285ab242e8c52ef9d7afe374c9485b122ae815f1707b875569d0433c1c3ce85"),
			MustHexID("c08397d5751b47bd3da044b908be0fb0e510d3149574dff7aeab33749b023bb171b5769990fe17469dbebc100bc150e798aeda426a2dcc766699a225fddd75c6"),
			MustHexID("0222c1c194b749736e593f937fad67ee348ac57287a15c7e42877aa38a9b87732a408bca370f812efd0eedbff13e6d5b854bf3ba1dec431a796ed47f32552b09"),
			MustHexID("03d859cd46ef02d9bfad5268461a6955426845eef4126de6be0fa4e8d7e0727ba2385b78f1a883a8239e95ebb814f2af8379632c7d5b100688eebc5841209582"),
			MustHexID("64d5004b7e043c39ff0bd10cb20094c287721d5251715884c280a612b494b3e9e1c64ba6f67614994c7d969a0d0c0295d107d53fc225d47c44c4b82852d6f960"),
			MustHexID("b0a5eefb2dab6f786670f35bf9641eefe6dd87fd3f1362bcab4aaa792903500ab23d88fae68411372e0813b057535a601d46e454323745a948017f6063a47b1f"),
			MustHexID("0cc6df0a3433d448b5684d2a3ffa9d1a825388177a18f44ad0008c7bd7702f1ec0fc38b83506f7de689c3b6ecb552599927e29699eed6bb867ff08f80068b287"),
			MustHexID("50772f7b8c03a4e153355fbbf79c8a80cf32af656ff0c7873c99911099d04a0dae0674706c357e0145ad017a0ade65e6052cb1b0d574fcd6f67da3eee0ace66b"),
			MustHexID("1ae37829c9ef41f8b508b82259ebac76b1ed900d7a45c08b7970f25d2d48ddd1829e2f11423a18749940b6dab8598c6e416cef0efd47e46e51f29a0bc65b37cd"),
			MustHexID("ba973cab31c2af091fc1644a93527d62b2394999e2b6ccbf158dd5ab9796a43d408786f1803ef4e29debfeb62fce2b6caa5ab2b24d1549c822a11c40c2856665"),
			MustHexID("bc413ad270dd6ea25bddba78f3298b03b8ba6f8608ac03d06007d4116fa78ef5a0cfe8c80155089382fc7a193243ee5500082660cb5d7793f60f2d7d18650964"),
			MustHexID("5a6a9ef07634d9eec3baa87c997b529b92652afa11473dfee41ef7037d5c06e0ddb9fe842364462d79dd31cff8a59a1b8d5bc2b810dea1d4cbbd3beb80ecec83"),
			MustHexID("f492c6ee2696d5f682f7f537757e52744c2ae560f1090a07024609e903d334e9e174fc01609c5a229ddbcac36c9d21adaf6457dab38a25bfd44f2f0ee4277998"),
			MustHexID("459e4db99298cb0467a90acee6888b08bb857450deac11015cced5104853be5adce5b69c740968bc7f931495d671a70cad9f48546d7cd203357fe9af0e8d2164"),
		},
		256: []NodeID{
			MustHexID("a8593af8a4aef7b806b5197612017951bac8845a1917ca9a6a15dd6086d608505144990b245785c4cd2d67a295701c7aac2aa18823fb0033987284b019656268"),
			MustHexID("d2eebef914928c3aad77fc1b2a495f52d2294acf5edaa7d8a530b540f094b861a68fe8348a46a7c302f08ab609d85912a4968eacfea0740847b29421b4795d9e"),
			MustHexID("b14bfcb31495f32b650b63cf7d08492e3e29071fdc73cf2da0da48d4b191a70ba1a65f42ad8c343206101f00f8a48e8db4b08bf3f622c0853e7323b250835b91"),
			MustHexID("7feaee0d818c03eb30e4e0bf03ade0f3c21ca38e938a761aa1781cf70bda8cc5cd631a6cc53dd44f1d4a6d3e2dae6513c6c66ee50cb2f0e9ad6f7e319b309fd9"),
			MustHexID("4ca3b657b139311db8d583c25dd5963005e46689e1317620496cc64129c7f3e52870820e0ec7941d28809311df6db8a2867bbd4f235b4248af24d7a9c22d1232"),
			MustHexID("1181defb1d16851d42dd951d84424d6bd1479137f587fa184d5a8152be6b6b16ed08bcdb2c2ed8539bcde98c80c432875f9f724737c316a2bd385a39d3cab1d8"),
			MustHexID("d9dd818769fa0c3ec9f553c759b92476f082817252a04a47dc1777740b1731d280058c66f982812f173a294acf4944a85ba08346e2de153ba3ba41ce8a62cb64"),
			MustHexID("bd7c4f8a9e770aa915c771b15e107ca123d838762da0d3ffc53aa6b53e9cd076cffc534ec4d2e4c334c683f1f5ea72e0e123f6c261915ed5b58ac1b59f003d88"),
			MustHexID("3dd5739c73649d510456a70e9d6b46a855864a4a3f744e088fd8c8da11b18e4c9b5f2d7da50b1c147b2bae5ca9609ae01f7a3cdea9dce34f80a91d29cd82f918"),
			MustHexID("f0d7df1efc439b4bcc0b762118c1cfa99b2a6143a9f4b10e3c9465125f4c9fca4ab88a2504169bbcad65492cf2f50da9dd5d077c39574a944f94d8246529066b"),
			MustHexID("dd598b9ba441448e5fb1a6ec6c5f5aa9605bad6e223297c729b1705d11d05f6bfd3d41988b694681ae69bb03b9a08bff4beab5596503d12a39bffb5cd6e94c7c"),
			MustHexID("3fce284ac97e567aebae681b15b7a2b6df9d873945536335883e4bbc26460c064370537f323fd1ada828ea43154992d14ac0cec0940a2bd2a3f42ec156d60c83"),
			MustHexID("7c8dfa8c1311cb14fb29a8ac11bca23ecc115e56d9fcf7b7ac1db9066aa4eb39f8b1dabf46e192a65be95ebfb4e839b5ab4533fef414921825e996b210dd53bd"),
			MustHexID("cafa6934f82120456620573d7f801390ed5e16ed619613a37e409e44ab355ef755e83565a913b48a9466db786f8d4fbd590bfec474c2524d4a2608d4eafd6abd"),
			MustHexID("9d16600d0dd310d77045769fed2cb427f32db88cd57d86e49390c2ba8a9698cfa856f775be2013237226e7bf47b248871cf865d23015937d1edeb20db5e3e760"),
			MustHexID("17be6b6ba54199b1d80eff866d348ea11d8a4b341d63ad9a6681d3ef8a43853ac564d153eb2a8737f0afc9ab320f6f95c55aa11aaa13bbb1ff422fd16bdf8188"),
		},
	},
}

type preminedTestnet struct {
	target    NodeID
	targetSha common.Hash // sha3(target)
	dists     [hashBits + 1][]NodeID
}

func (tn *preminedTestnet) findnode(toid NodeID, toaddr *net.UDPAddr, target NodeID) ([]*Node, error) {
	// current log distance is encoded in port number
	// fmt.Println("findnode query at dist", toaddr.Port)
	if toaddr.Port == 0 {
		panic("query to node at distance 0")
	}
	next := uint16(toaddr.Port) - 1
	var result []*Node
	for i, id := range tn.dists[toaddr.Port] {
		result = append(result, NewNode(id, net.ParseIP("127.0.0.1"), next, uint16(i)))
	}
	return result, nil
}

func (*preminedTestnet) close()                                      {}
func (*preminedTestnet) waitping(from NodeID) error                  { return nil }
func (*preminedTestnet) ping(toid NodeID, toaddr *net.UDPAddr) error { return nil }

// mine generates a testnet struct literal with nodes at
// various distances to the given target.
func (n *preminedTestnet) mine(target NodeID) {
	n.target = target
	n.targetSha = crypto.Keccak256Hash(n.target[:])
	found := 0
	for found < bucketSize*10 {
		k := newkey()
		id := PubkeyID(&k.PublicKey)
		sha := crypto.Keccak256Hash(id[:])
		ld := logdist(n.targetSha, sha)
		if len(n.dists[ld]) < bucketSize {
			n.dists[ld] = append(n.dists[ld], id)
			fmt.Println("found ID with ld", ld)
			found++
		}
	}
	fmt.Println("&preminedTestnet{")
	fmt.Printf("	target: %#v,\n", n.target)
	fmt.Printf("	targetSha: %#v,\n", n.targetSha)
	fmt.Printf("	dists: [%d][]NodeID{\n", len(n.dists))
	for ld, ns := range n.dists {
		if len(ns) == 0 {
			continue
		}
		fmt.Printf("		%d: []NodeID{\n", ld)
		for _, n := range ns {
			fmt.Printf("			MustHexID(\"%x\"),\n", n[:])
		}
		fmt.Println("		},")
	}
	fmt.Println("	},")
	fmt.Println("}")
}

func hasDuplicates(slice []*Node) bool {
	seen := make(map[NodeID]bool)
	for i, e := range slice {
		if e == nil {
			panic(fmt.Sprintf("nil *Node at %d", i))
		}
		if seen[e.ID] {
			return true
		}
		seen[e.ID] = true
	}
	return false
}

func sortedByDistanceTo(distbase common.Hash, slice []*Node) bool {
	var last common.Hash
	for i, e := range slice {
		if i > 0 && distcmp(distbase, e.sha, last) < 0 {
			return false
		}
		last = e.sha
	}
	return true
}

func contains(ns []*Node, id NodeID) bool {
	for _, n := range ns {
		if n.ID == id {
			return true
		}
	}
	return false
}

// gen wraps quick.Value so it's easier to use.
// it generates a random value of the given value's type.
func gen(typ interface{}, rand *rand.Rand) interface{} {
	v, ok := quick.Value(reflect.TypeOf(typ), rand)
	if !ok {
		panic(fmt.Sprintf("couldn't generate random value of type %T", typ))
	}
	return v.Interface()
}

func newkey() *ecdsa.PrivateKey {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic("couldn't generate key: " + err.Error())
	}
	return key
}
