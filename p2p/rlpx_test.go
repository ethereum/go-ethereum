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

package p2p

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestSharedSecret(t *testing.T) {
	prv0, _ := crypto.GenerateKey() // = ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	pub0 := &prv0.PublicKey
	prv1, _ := crypto.GenerateKey()
	pub1 := &prv1.PublicKey

	ss0, err := ecies.ImportECDSA(prv0).GenerateShared(ecies.ImportECDSAPublic(pub1), sskLen, sskLen)
	if err != nil {
		return
	}
	ss1, err := ecies.ImportECDSA(prv1).GenerateShared(ecies.ImportECDSAPublic(pub0), sskLen, sskLen)
	if err != nil {
		return
	}
	t.Logf("Secret:\n%v %x\n%v %x", len(ss0), ss0, len(ss0), ss1)
	if !bytes.Equal(ss0, ss1) {
		t.Errorf("dont match :(")
	}
}

func TestEncHandshake(t *testing.T) {
	for i := 0; i < 10; i++ {
		start := time.Now()
		if err := testEncHandshake(nil); err != nil {
			t.Fatalf("i=%d %v", i, err)
		}
		t.Logf("(without token) %d %v\n", i+1, time.Since(start))
	}
	for i := 0; i < 10; i++ {
		tok := make([]byte, shaLen)
		rand.Reader.Read(tok)
		start := time.Now()
		if err := testEncHandshake(tok); err != nil {
			t.Fatalf("i=%d %v", i, err)
		}
		t.Logf("(with token) %d %v\n", i+1, time.Since(start))
	}
}

func testEncHandshake(token []byte) error {
	type result struct {
		side string
		id   discover.NodeID
		err  error
	}
	var (
		prv0, _  = crypto.GenerateKey()
		prv1, _  = crypto.GenerateKey()
		fd0, fd1 = net.Pipe()
		c0, c1   = newRLPX(fd0).(*rlpx), newRLPX(fd1).(*rlpx)
		output   = make(chan result)
	)

	go func() {
		r := result{side: "initiator"}
		defer func() { output <- r }()
		defer fd0.Close()

		dest := &discover.Node{ID: discover.PubkeyID(&prv1.PublicKey)}
		r.id, r.err = c0.doEncHandshake(prv0, dest)
		if r.err != nil {
			return
		}
		id1 := discover.PubkeyID(&prv1.PublicKey)
		if r.id != id1 {
			r.err = fmt.Errorf("remote ID mismatch: got %v, want: %v", r.id, id1)
		}
	}()
	go func() {
		r := result{side: "receiver"}
		defer func() { output <- r }()
		defer fd1.Close()

		r.id, r.err = c1.doEncHandshake(prv1, nil)
		if r.err != nil {
			return
		}
		id0 := discover.PubkeyID(&prv0.PublicKey)
		if r.id != id0 {
			r.err = fmt.Errorf("remote ID mismatch: got %v, want: %v", r.id, id0)
		}
	}()

	// wait for results from both sides
	r1, r2 := <-output, <-output
	if r1.err != nil {
		return fmt.Errorf("%s side error: %v", r1.side, r1.err)
	}
	if r2.err != nil {
		return fmt.Errorf("%s side error: %v", r2.side, r2.err)
	}

	// compare derived secrets
	if !reflect.DeepEqual(c0.rw.egressMAC, c1.rw.ingressMAC) {
		return fmt.Errorf("egress mac mismatch:\n c0.rw: %#v\n c1.rw: %#v", c0.rw.egressMAC, c1.rw.ingressMAC)
	}
	if !reflect.DeepEqual(c0.rw.ingressMAC, c1.rw.egressMAC) {
		return fmt.Errorf("ingress mac mismatch:\n c0.rw: %#v\n c1.rw: %#v", c0.rw.ingressMAC, c1.rw.egressMAC)
	}
	if !reflect.DeepEqual(c0.rw.enc, c1.rw.enc) {
		return fmt.Errorf("enc cipher mismatch:\n c0.rw: %#v\n c1.rw: %#v", c0.rw.enc, c1.rw.enc)
	}
	if !reflect.DeepEqual(c0.rw.dec, c1.rw.dec) {
		return fmt.Errorf("dec cipher mismatch:\n c0.rw: %#v\n c1.rw: %#v", c0.rw.dec, c1.rw.dec)
	}
	return nil
}

func TestProtocolHandshake(t *testing.T) {
	var (
		prv0, _ = crypto.GenerateKey()
		node0   = &discover.Node{ID: discover.PubkeyID(&prv0.PublicKey), IP: net.IP{1, 2, 3, 4}, TCP: 33}
		hs0     = &protoHandshake{Version: 3, ID: node0.ID, Caps: []Cap{{"a", 0}, {"b", 2}}}

		prv1, _ = crypto.GenerateKey()
		node1   = &discover.Node{ID: discover.PubkeyID(&prv1.PublicKey), IP: net.IP{5, 6, 7, 8}, TCP: 44}
		hs1     = &protoHandshake{Version: 3, ID: node1.ID, Caps: []Cap{{"c", 1}, {"d", 3}}}

		fd0, fd1 = net.Pipe()
		wg       sync.WaitGroup
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		defer fd1.Close()
		rlpx := newRLPX(fd0)
		remid, err := rlpx.doEncHandshake(prv0, node1)
		if err != nil {
			t.Errorf("dial side enc handshake failed: %v", err)
			return
		}
		if remid != node1.ID {
			t.Errorf("dial side remote id mismatch: got %v, want %v", remid, node1.ID)
			return
		}

		phs, err := rlpx.doProtoHandshake(hs0)
		if err != nil {
			t.Errorf("dial side proto handshake error: %v", err)
			return
		}
		phs.Rest = nil
		if !reflect.DeepEqual(phs, hs1) {
			t.Errorf("dial side proto handshake mismatch:\ngot: %s\nwant: %s\n", spew.Sdump(phs), spew.Sdump(hs1))
			return
		}
		rlpx.close(DiscQuitting)
	}()
	go func() {
		defer wg.Done()
		defer fd1.Close()
		rlpx := newRLPX(fd1)
		remid, err := rlpx.doEncHandshake(prv1, nil)
		if err != nil {
			t.Errorf("listen side enc handshake failed: %v", err)
			return
		}
		if remid != node0.ID {
			t.Errorf("listen side remote id mismatch: got %v, want %v", remid, node0.ID)
			return
		}

		phs, err := rlpx.doProtoHandshake(hs1)
		if err != nil {
			t.Errorf("listen side proto handshake error: %v", err)
			return
		}
		phs.Rest = nil
		if !reflect.DeepEqual(phs, hs0) {
			t.Errorf("listen side proto handshake mismatch:\ngot: %s\nwant: %s\n", spew.Sdump(phs), spew.Sdump(hs0))
			return
		}

		if err := ExpectMsg(rlpx, discMsg, []DiscReason{DiscQuitting}); err != nil {
			t.Errorf("error receiving disconnect: %v", err)
		}
	}()
	wg.Wait()
}

func TestProtocolHandshakeErrors(t *testing.T) {
	our := &protoHandshake{Version: 3, Caps: []Cap{{"foo", 2}, {"bar", 3}}, Name: "quux"}
	tests := []struct {
		code uint64
		msg  interface{}
		err  error
	}{
		{
			code: discMsg,
			msg:  []DiscReason{DiscQuitting},
			err:  DiscQuitting,
		},
		{
			code: 0x989898,
			msg:  []byte{1},
			err:  errors.New("expected handshake, got 989898"),
		},
		{
			code: handshakeMsg,
			msg:  make([]byte, baseProtocolMaxMsgSize+2),
			err:  errors.New("message too big"),
		},
		{
			code: handshakeMsg,
			msg:  []byte{1, 2, 3},
			err:  newPeerError(errInvalidMsg, "(code 0) (size 4) rlp: expected input list for p2p.protoHandshake"),
		},
		{
			code: handshakeMsg,
			msg:  &protoHandshake{Version: 3},
			err:  DiscInvalidIdentity,
		},
	}

	for i, test := range tests {
		p1, p2 := MsgPipe()
		go Send(p1, test.code, test.msg)
		_, err := readProtocolHandshake(p2, our)
		if !reflect.DeepEqual(err, test.err) {
			t.Errorf("test %d: error mismatch: got %q, want %q", i, err, test.err)
		}
	}
}

func TestRLPXFrameFake(t *testing.T) {
	buf := new(bytes.Buffer)
	hash := fakeHash([]byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1})
	rw := newRLPXFrameRW(buf, secrets{
		AES:        crypto.Sha3(),
		MAC:        crypto.Sha3(),
		IngressMAC: hash,
		EgressMAC:  hash,
	})

	golden := unhex(`
00828ddae471818bb0bfa6b551d1cb42
01010101010101010101010101010101
ba628a4ba590cb43f7848f41c4382885
01010101010101010101010101010101
`)

	// Check WriteMsg. This puts a message into the buffer.
	if err := Send(rw, 8, []uint{1, 2, 3, 4}); err != nil {
		t.Fatalf("WriteMsg error: %v", err)
	}
	written := buf.Bytes()
	if !bytes.Equal(written, golden) {
		t.Fatalf("output mismatch:\n  got:  %x\n  want: %x", written, golden)
	}

	// Check ReadMsg. It reads the message encoded by WriteMsg, which
	// is equivalent to the golden message above.
	msg, err := rw.ReadMsg()
	if err != nil {
		t.Fatalf("ReadMsg error: %v", err)
	}
	if msg.Size != 5 {
		t.Errorf("msg size mismatch: got %d, want %d", msg.Size, 5)
	}
	if msg.Code != 8 {
		t.Errorf("msg code mismatch: got %d, want %d", msg.Code, 8)
	}
	payload, _ := ioutil.ReadAll(msg.Payload)
	wantPayload := unhex("C401020304")
	if !bytes.Equal(payload, wantPayload) {
		t.Errorf("msg payload mismatch:\ngot  %x\nwant %x", payload, wantPayload)
	}
}

type fakeHash []byte

func (fakeHash) Write(p []byte) (int, error) { return len(p), nil }
func (fakeHash) Reset()                      {}
func (fakeHash) BlockSize() int              { return 0 }

func (h fakeHash) Size() int           { return len(h) }
func (h fakeHash) Sum(b []byte) []byte { return append(b, h...) }

func TestRLPXFrameRW(t *testing.T) {
	var (
		aesSecret      = make([]byte, 16)
		macSecret      = make([]byte, 16)
		egressMACinit  = make([]byte, 32)
		ingressMACinit = make([]byte, 32)
	)
	for _, s := range [][]byte{aesSecret, macSecret, egressMACinit, ingressMACinit} {
		rand.Read(s)
	}
	conn := new(bytes.Buffer)

	s1 := secrets{
		AES:        aesSecret,
		MAC:        macSecret,
		EgressMAC:  sha3.NewKeccak256(),
		IngressMAC: sha3.NewKeccak256(),
	}
	s1.EgressMAC.Write(egressMACinit)
	s1.IngressMAC.Write(ingressMACinit)
	rw1 := newRLPXFrameRW(conn, s1)

	s2 := secrets{
		AES:        aesSecret,
		MAC:        macSecret,
		EgressMAC:  sha3.NewKeccak256(),
		IngressMAC: sha3.NewKeccak256(),
	}
	s2.EgressMAC.Write(ingressMACinit)
	s2.IngressMAC.Write(egressMACinit)
	rw2 := newRLPXFrameRW(conn, s2)

	// send some messages
	for i := 0; i < 10; i++ {
		// write message into conn buffer
		wmsg := []interface{}{"foo", "bar", strings.Repeat("test", i)}
		err := Send(rw1, uint64(i), wmsg)
		if err != nil {
			t.Fatalf("WriteMsg error (i=%d): %v", i, err)
		}

		// read message that rw1 just wrote
		msg, err := rw2.ReadMsg()
		if err != nil {
			t.Fatalf("ReadMsg error (i=%d): %v", i, err)
		}
		if msg.Code != uint64(i) {
			t.Fatalf("msg code mismatch: got %d, want %d", msg.Code, i)
		}
		payload, _ := ioutil.ReadAll(msg.Payload)
		wantPayload, _ := rlp.EncodeToBytes(wmsg)
		if !bytes.Equal(payload, wantPayload) {
			t.Fatalf("msg payload mismatch:\ngot  %x\nwant %x", payload, wantPayload)
		}
	}
}

type handshakeAuthTest struct {
	input       string
	isPlain     bool
	wantVersion uint
	wantRest    []rlp.RawValue
}

var eip8HandshakeAuthTests = []handshakeAuthTest{
	// (Auth₁) RLPx v4 plain encoding
	{
		input: `
			048ca79ad18e4b0659fab4853fe5bc58eb83992980f4c9cc147d2aa31532efd29a3d3dc6a3d89eaf
			913150cfc777ce0ce4af2758bf4810235f6e6ceccfee1acc6b22c005e9e3a49d6448610a58e98744
			ba3ac0399e82692d67c1f58849050b3024e21a52c9d3b01d871ff5f210817912773e610443a9ef14
			2e91cdba0bd77b5fdf0769b05671fc35f83d83e4d3b0b000c6b2a1b1bba89e0fc51bf4e460df3105
			c444f14be226458940d6061c296350937ffd5e3acaceeaaefd3c6f74be8e23e0f45163cc7ebd7622
			0f0128410fd05250273156d548a414444ae2f7dea4dfca2d43c057adb701a715bf59f6fb66b2d1d2
			0f2c703f851cbf5ac47396d9ca65b6260bd141ac4d53e2de585a73d1750780db4c9ee4cd4d225173
			a4592ee77e2bd94d0be3691f3b406f9bba9b591fc63facc016bfa8
		`,
		isPlain:     true,
		wantVersion: 4,
	},
	// (Auth₂) EIP-8 encoding
	{
		input: `
			01b304ab7578555167be8154d5cc456f567d5ba302662433674222360f08d5f1534499d3678b513b
			0fca474f3a514b18e75683032eb63fccb16c156dc6eb2c0b1593f0d84ac74f6e475f1b8d56116b84
			9634a8c458705bf83a626ea0384d4d7341aae591fae42ce6bd5c850bfe0b999a694a49bbbaf3ef6c
			da61110601d3b4c02ab6c30437257a6e0117792631a4b47c1d52fc0f8f89caadeb7d02770bf999cc
			147d2df3b62e1ffb2c9d8c125a3984865356266bca11ce7d3a688663a51d82defaa8aad69da39ab6
			d5470e81ec5f2a7a47fb865ff7cca21516f9299a07b1bc63ba56c7a1a892112841ca44b6e0034dee
			70c9adabc15d76a54f443593fafdc3b27af8059703f88928e199cb122362a4b35f62386da7caad09
			c001edaeb5f8a06d2b26fb6cb93c52a9fca51853b68193916982358fe1e5369e249875bb8d0d0ec3
			6f917bc5e1eafd5896d46bd61ff23f1a863a8a8dcd54c7b109b771c8e61ec9c8908c733c0263440e
			2aa067241aaa433f0bb053c7b31a838504b148f570c0ad62837129e547678c5190341e4f1693956c
			3bf7678318e2d5b5340c9e488eefea198576344afbdf66db5f51204a6961a63ce072c8926c
		`,
		wantVersion: 4,
		wantRest:    []rlp.RawValue{},
	},
	// (Auth₃) RLPx v4 EIP-8 encoding with version 56, additional list elements
	{
		input: `
			01b8044c6c312173685d1edd268aa95e1d495474c6959bcdd10067ba4c9013df9e40ff45f5bfd6f7
			2471f93a91b493f8e00abc4b80f682973de715d77ba3a005a242eb859f9a211d93a347fa64b597bf
			280a6b88e26299cf263b01b8dfdb712278464fd1c25840b995e84d367d743f66c0e54a586725b7bb
			f12acca27170ae3283c1073adda4b6d79f27656993aefccf16e0d0409fe07db2dc398a1b7e8ee93b
			cd181485fd332f381d6a050fba4c7641a5112ac1b0b61168d20f01b479e19adf7fdbfa0905f63352
			bfc7e23cf3357657455119d879c78d3cf8c8c06375f3f7d4861aa02a122467e069acaf513025ff19
			6641f6d2810ce493f51bee9c966b15c5043505350392b57645385a18c78f14669cc4d960446c1757
			1b7c5d725021babbcd786957f3d17089c084907bda22c2b2675b4378b114c601d858802a55345a15
			116bc61da4193996187ed70d16730e9ae6b3bb8787ebcaea1871d850997ddc08b4f4ea668fbf3740
			7ac044b55be0908ecb94d4ed172ece66fd31bfdadf2b97a8bc690163ee11f5b575a4b44e36e2bfb2
			f0fce91676fd64c7773bac6a003f481fddd0bae0a1f31aa27504e2a533af4cef3b623f4791b2cca6
			d490
		`,
		wantVersion: 56,
		wantRest:    []rlp.RawValue{{0x01}, {0x02}, {0xC2, 0x04, 0x05}},
	},
}

type handshakeAckTest struct {
	input       string
	wantVersion uint
	wantRest    []rlp.RawValue
}

var eip8HandshakeRespTests = []handshakeAckTest{
	// (Ack₁) RLPx v4 plain encoding
	{
		input: `
			049f8abcfa9c0dc65b982e98af921bc0ba6e4243169348a236abe9df5f93aa69d99cadddaa387662
			b0ff2c08e9006d5a11a278b1b3331e5aaabf0a32f01281b6f4ede0e09a2d5f585b26513cb794d963
			5a57563921c04a9090b4f14ee42be1a5461049af4ea7a7f49bf4c97a352d39c8d02ee4acc416388c
			1c66cec761d2bc1c72da6ba143477f049c9d2dde846c252c111b904f630ac98e51609b3b1f58168d
			dca6505b7196532e5f85b259a20c45e1979491683fee108e9660edbf38f3add489ae73e3dda2c71b
			d1497113d5c755e942d1
		`,
		wantVersion: 4,
	},
	// (Ack₂) EIP-8 encoding
	{
		input: `
			01ea0451958701280a56482929d3b0757da8f7fbe5286784beead59d95089c217c9b917788989470
			b0e330cc6e4fb383c0340ed85fab836ec9fb8a49672712aeabbdfd1e837c1ff4cace34311cd7f4de
			05d59279e3524ab26ef753a0095637ac88f2b499b9914b5f64e143eae548a1066e14cd2f4bd7f814
			c4652f11b254f8a2d0191e2f5546fae6055694aed14d906df79ad3b407d94692694e259191cde171
			ad542fc588fa2b7333313d82a9f887332f1dfc36cea03f831cb9a23fea05b33deb999e85489e645f
			6aab1872475d488d7bd6c7c120caf28dbfc5d6833888155ed69d34dbdc39c1f299be1057810f34fb
			e754d021bfca14dc989753d61c413d261934e1a9c67ee060a25eefb54e81a4d14baff922180c395d
			3f998d70f46f6b58306f969627ae364497e73fc27f6d17ae45a413d322cb8814276be6ddd13b885b
			201b943213656cde498fa0e9ddc8e0b8f8a53824fbd82254f3e2c17e8eaea009c38b4aa0a3f306e8
			797db43c25d68e86f262e564086f59a2fc60511c42abfb3057c247a8a8fe4fb3ccbadde17514b7ac
			8000cdb6a912778426260c47f38919a91f25f4b5ffb455d6aaaf150f7e5529c100ce62d6d92826a7
			1778d809bdf60232ae21ce8a437eca8223f45ac37f6487452ce626f549b3b5fdee26afd2072e4bc7
			5833c2464c805246155289f4
		`,
		wantVersion: 4,
		wantRest:    []rlp.RawValue{},
	},
	// (Ack₃) EIP-8 encoding with version 57, additional list elements
	{
		input: `
			01f004076e58aae772bb101ab1a8e64e01ee96e64857ce82b1113817c6cdd52c09d26f7b90981cd7
			ae835aeac72e1573b8a0225dd56d157a010846d888dac7464baf53f2ad4e3d584531fa203658fab0
			3a06c9fd5e35737e417bc28c1cbf5e5dfc666de7090f69c3b29754725f84f75382891c561040ea1d
			dc0d8f381ed1b9d0d4ad2a0ec021421d847820d6fa0ba66eaf58175f1b235e851c7e2124069fbc20
			2888ddb3ac4d56bcbd1b9b7eab59e78f2e2d400905050f4a92dec1c4bdf797b3fc9b2f8e84a482f3
			d800386186712dae00d5c386ec9387a5e9c9a1aca5a573ca91082c7d68421f388e79127a5177d4f8
			590237364fd348c9611fa39f78dcdceee3f390f07991b7b47e1daa3ebcb6ccc9607811cb17ce51f1
			c8c2c5098dbdd28fca547b3f58c01a424ac05f869f49c6a34672ea2cbbc558428aa1fe48bbfd6115
			8b1b735a65d99f21e70dbc020bfdface9f724a0d1fb5895db971cc81aa7608baa0920abb0a565c9c
			436e2fd13323428296c86385f2384e408a31e104670df0791d93e743a3a5194ee6b076fb6323ca59
			3011b7348c16cf58f66b9633906ba54a2ee803187344b394f75dd2e663a57b956cb830dd7a908d4f
			39a2336a61ef9fda549180d4ccde21514d117b6c6fd07a9102b5efe710a32af4eeacae2cb3b1dec0
			35b9593b48b9d3ca4c13d245d5f04169b0b1
		`,
		wantVersion: 57,
		wantRest:    []rlp.RawValue{{0x06}, {0xC2, 0x07, 0x08}, {0x81, 0xFA}},
	},
}

func TestHandshakeForwardCompatibility(t *testing.T) {
	var (
		keyA, _       = crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
		keyB, _       = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		pubA          = crypto.FromECDSAPub(&keyA.PublicKey)[1:]
		pubB          = crypto.FromECDSAPub(&keyB.PublicKey)[1:]
		ephA, _       = crypto.HexToECDSA("869d6ecf5211f1cc60418a13b9d870b22959d0c16f02bec714c960dd2298a32d")
		ephB, _       = crypto.HexToECDSA("e238eb8e04fee6511ab04c6dd3c89ce097b11f25d584863ac2b6d5b35b1847e4")
		ephPubA       = crypto.FromECDSAPub(&ephA.PublicKey)[1:]
		ephPubB       = crypto.FromECDSAPub(&ephB.PublicKey)[1:]
		nonceA        = unhex("7e968bba13b6c50e2c4cd7f241cc0d64d1ac25c7f5952df231ac6a2bda8ee5d6")
		nonceB        = unhex("559aead08264d5795d3909718cdd05abd49572e84fe55590eef31a88a08fdffd")
		_, _, _, _    = pubA, pubB, ephPubA, ephPubB
		authSignature = unhex("299ca6acfd35e3d72d8ba3d1e2b60b5561d5af5218eb5bc182045769eb4226910a301acae3b369fffc4a4899d6b02531e89fd4fe36a2cf0d93607ba470b50f7800")
		_             = authSignature
	)
	makeAuth := func(test handshakeAuthTest) *authMsgV4 {
		msg := &authMsgV4{Version: test.wantVersion, Rest: test.wantRest, gotPlain: test.isPlain}
		copy(msg.Signature[:], authSignature)
		copy(msg.InitiatorPubkey[:], pubA)
		copy(msg.Nonce[:], nonceA)
		return msg
	}
	makeAck := func(test handshakeAckTest) *authRespV4 {
		msg := &authRespV4{Version: test.wantVersion, Rest: test.wantRest}
		copy(msg.RandomPubkey[:], ephPubB)
		copy(msg.Nonce[:], nonceB)
		return msg
	}

	// check auth msg parsing
	for _, test := range eip8HandshakeAuthTests {
		r := bytes.NewReader(unhex(test.input))
		msg := new(authMsgV4)
		ciphertext, err := readHandshakeMsg(msg, encAuthMsgLen, keyB, r)
		if err != nil {
			t.Errorf("error for input %x:\n  %v", unhex(test.input), err)
			continue
		}
		if !bytes.Equal(ciphertext, unhex(test.input)) {
			t.Errorf("wrong ciphertext for input %x:\n  %x", unhex(test.input), ciphertext)
		}
		want := makeAuth(test)
		if !reflect.DeepEqual(msg, want) {
			t.Errorf("wrong msg for input %x:\ngot %s\nwant %s", unhex(test.input), spew.Sdump(msg), spew.Sdump(want))
		}
	}

	// check auth resp parsing
	for _, test := range eip8HandshakeRespTests {
		input := unhex(test.input)
		r := bytes.NewReader(input)
		msg := new(authRespV4)
		ciphertext, err := readHandshakeMsg(msg, encAuthRespLen, keyA, r)
		if err != nil {
			t.Errorf("error for input %x:\n  %v", input, err)
			continue
		}
		if !bytes.Equal(ciphertext, input) {
			t.Errorf("wrong ciphertext for input %x:\n  %x", input, err)
		}
		want := makeAck(test)
		if !reflect.DeepEqual(msg, want) {
			t.Errorf("wrong msg for input %x:\ngot %s\nwant %s", input, spew.Sdump(msg), spew.Sdump(want))
		}
	}

	// check derivation for (Auth₂, Ack₂) on recipient side
	var (
		hs = &encHandshake{
			initiator:     false,
			respNonce:     nonceB,
			randomPrivKey: ecies.ImportECDSA(ephB),
		}
		authCiphertext     = unhex(eip8HandshakeAuthTests[1].input)
		authRespCiphertext = unhex(eip8HandshakeRespTests[1].input)
		authMsg            = makeAuth(eip8HandshakeAuthTests[1])
		wantAES            = unhex("80e8632c05fed6fc2a13b0f8d31a3cf645366239170ea067065aba8e28bac487")
		wantMAC            = unhex("2ea74ec5dae199227dff1af715362700e989d889d7a493cb0639691efb8e5f98")
		wantFooIngressHash = unhex("0c7ec6340062cc46f5e9f1e3cf86f8c8c403c5a0964f5df0ebd34a75ddc86db5")
	)
	if err := hs.handleAuthMsg(authMsg, keyB); err != nil {
		t.Fatalf("handleAuthMsg: %v", err)
	}
	derived, err := hs.secrets(authCiphertext, authRespCiphertext)
	if err != nil {
		t.Fatalf("secrets: %v", err)
	}
	if !bytes.Equal(derived.AES, wantAES) {
		t.Errorf("aes-secret mismatch:\ngot %x\nwant %x", derived.AES, wantAES)
	}
	if !bytes.Equal(derived.MAC, wantMAC) {
		t.Errorf("mac-secret mismatch:\ngot %x\nwant %x", derived.MAC, wantMAC)
	}
	io.WriteString(derived.IngressMAC, "foo")
	fooIngressHash := derived.IngressMAC.Sum(nil)
	if !bytes.Equal(fooIngressHash, wantFooIngressHash) {
		t.Errorf("ingress-mac('foo') mismatch:\ngot %x\nwant %x", fooIngressHash, wantFooIngressHash)
	}
}
