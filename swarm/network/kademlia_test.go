// Copyright 2017 The go-ethereum Authors
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

package network

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/pot"
)

func init() {
	h := log.LvlFilterHandler(log.LvlWarn, log.StreamHandler(os.Stderr, log.TerminalFormat(true)))
	log.Root().SetHandler(h)
}

func testKadPeerAddr(s string) *BzzAddr {
	a := pot.NewAddressFromString(s)
	return &BzzAddr{OAddr: a, UAddr: a}
}

type testDropPeer struct {
	Peer
	dropc chan error
}

type dropError struct {
	error
	addr string
}

func (d *testDropPeer) Drop(err error) {
	err2 := &dropError{err, binStr(d)}
	d.dropc <- err2
}

type testKademlia struct {
	*Kademlia
	Discovery bool
	dropc     chan error
}

func newTestKademlia(b string) *testKademlia {
	params := NewKadParams()
	params.MinBinSize = 1
	params.MinProxBinSize = 2
	base := pot.NewAddressFromString(b)
	return &testKademlia{
		NewKademlia(base, params),
		false,
		make(chan error),
	}
}

func (k *testKademlia) newTestKadPeer(s string) Peer {
	return &testDropPeer{&BzzPeer{BzzAddr: testKadPeerAddr(s)}, k.dropc}
}

func (k *testKademlia) On(ons ...string) *testKademlia {
	for _, s := range ons {
		k.Kademlia.On(k.newTestKadPeer(s).(OverlayConn))
	}
	return k
}

func (k *testKademlia) Off(offs ...string) *testKademlia {
	for _, s := range offs {
		k.Kademlia.Off(k.newTestKadPeer(s).(OverlayConn))
	}

	return k
}

func (k *testKademlia) Register(regs ...string) *testKademlia {
	var as []OverlayAddr
	for _, s := range regs {
		as = append(as, testKadPeerAddr(s))
	}
	err := k.Kademlia.Register(as)
	if err != nil {
		panic(err.Error())
	}
	return k
}

func testSuggestPeer(t *testing.T, k *testKademlia, expAddr string, expPo int, expWant bool) error {
	addr, o, want := k.SuggestPeer()
	if binStr(addr) != expAddr {
		return fmt.Errorf("incorrect peer address suggested. expected %v, got %v", expAddr, binStr(addr))
	}
	if o != expPo {
		return fmt.Errorf("incorrect prox order suggested. expected %v, got %v", expPo, o)
	}
	if want != expWant {
		return fmt.Errorf("expected SuggestPeer to want peers: %v", expWant)
	}
	return nil
}

func binStr(a OverlayPeer) string {
	if a == nil {
		return "<nil>"
	}
	return pot.ToBin(a.Address())[:8]
}

func TestSuggestPeerBug(t *testing.T) {
	// 2 row gap, unsaturated proxbin, no callables -> want PO 0
	k := newTestKademlia("00000000").On(
		"10000000", "11000000",
		"01000000",

		"00010000", "00011000",
	).Off(
		"01000000",
	)
	err := testSuggestPeer(t, k, "01000000", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestSuggestPeerFindPeers(t *testing.T) {
	// 2 row gap, unsaturated proxbin, no callables -> want PO 0
	k := newTestKademlia("00000000").On("00100000")
	err := testSuggestPeer(t, k, "<nil>", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	// 2 row gap, saturated proxbin, no callables -> want PO 0
	k.On("00010000")
	err = testSuggestPeer(t, k, "<nil>", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	// 1 row gap (1 less), saturated proxbin, no callables -> want PO 1
	k.On("10000000")
	err = testSuggestPeer(t, k, "<nil>", 1, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	// no gap (1 less), saturated proxbin, no callables -> do not want more
	k.On("01000000", "00100001")
	err = testSuggestPeer(t, k, "<nil>", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	// oversaturated proxbin, > do not want more
	k.On("00100001")
	err = testSuggestPeer(t, k, "<nil>", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	// reintroduce gap, disconnected peer callable
	// log.Info(k.String())
	k.Off("01000000")
	err = testSuggestPeer(t, k, "01000000", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	// second time disconnected peer not callable
	// with reasonably set Interval
	err = testSuggestPeer(t, k, "<nil>", 1, true)
	if err != nil {
		t.Fatal(err.Error())
	}

	// on and off again, peer callable again
	k.On("01000000")
	k.Off("01000000")
	err = testSuggestPeer(t, k, "01000000", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	k.On("01000000")
	// new closer peer appears, it is immediately wanted
	k.Register("00010001")
	err = testSuggestPeer(t, k, "00010001", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	// PO1 disconnects
	k.On("00010001")
	log.Info(k.String())
	k.Off("01000000")
	log.Info(k.String())
	// second time, gap filling
	err = testSuggestPeer(t, k, "01000000", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	k.On("01000000")
	err = testSuggestPeer(t, k, "<nil>", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	k.MinBinSize = 2
	err = testSuggestPeer(t, k, "<nil>", 0, true)
	if err != nil {
		t.Fatal(err.Error())
	}

	k.Register("01000001")
	err = testSuggestPeer(t, k, "01000001", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	k.On("10000001")
	log.Trace(fmt.Sprintf("Kad:\n%v", k.String()))
	err = testSuggestPeer(t, k, "<nil>", 1, true)
	if err != nil {
		t.Fatal(err.Error())
	}

	k.On("01000001")
	err = testSuggestPeer(t, k, "<nil>", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	k.MinBinSize = 3
	k.Register("10000010")
	err = testSuggestPeer(t, k, "10000010", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	k.On("10000010")
	err = testSuggestPeer(t, k, "<nil>", 1, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	k.On("01000010")
	err = testSuggestPeer(t, k, "<nil>", 2, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	k.On("00100010")
	err = testSuggestPeer(t, k, "<nil>", 3, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	k.On("00010010")
	err = testSuggestPeer(t, k, "<nil>", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

}

func TestSuggestPeerRetries(t *testing.T) {
	t.Skip("Test is disabled, because it is flaky. It fails with kademlia_test.go:346: incorrect peer address suggested. expected <nil>, got 01000000")
	// 2 row gap, unsaturated proxbin, no callables -> want PO 0
	k := newTestKademlia("00000000")
	k.RetryInterval = int64(100 * time.Millisecond) // cycle
	k.MaxRetries = 50
	k.RetryExponent = 2
	sleep := func(n int) {
		ts := k.RetryInterval
		for i := 1; i < n; i++ {
			ts *= int64(k.RetryExponent)
		}
		time.Sleep(time.Duration(ts))
	}

	k.Register("01000000")
	k.On("00000001", "00000010")
	err := testSuggestPeer(t, k, "01000000", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = testSuggestPeer(t, k, "<nil>", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	sleep(1)
	err = testSuggestPeer(t, k, "01000000", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = testSuggestPeer(t, k, "<nil>", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	sleep(1)
	err = testSuggestPeer(t, k, "01000000", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = testSuggestPeer(t, k, "<nil>", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	sleep(2)
	err = testSuggestPeer(t, k, "01000000", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = testSuggestPeer(t, k, "<nil>", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	sleep(2)
	err = testSuggestPeer(t, k, "<nil>", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

}

func TestKademliaHiveString(t *testing.T) {
	k := newTestKademlia("00000000").On("01000000", "00100000").Register("10000000", "10000001")
	k.MaxProxDisplay = 8
	h := k.String()
	expH := "\n=========================================================================\nMon Feb 27 12:10:28 UTC 2017 KΛÐΞMLIΛ hive: queen's address: 000000\npopulation: 2 (4), MinProxBinSize: 2, MinBinSize: 1, MaxBinSize: 4\n000  0                              |  2 8100 (0) 8000 (0)\n============ DEPTH: 1 ==========================================\n001  1 4000                         |  1 4000 (0)\n002  1 2000                         |  1 2000 (0)\n003  0                              |  0\n004  0                              |  0\n005  0                              |  0\n006  0                              |  0\n007  0                              |  0\n========================================================================="
	if expH[104:] != h[104:] {
		t.Fatalf("incorrect hive output. expected %v, got %v", expH, h)
	}
}

// testKademliaCase constructs the kademlia and PeerPot map to validate
// the SuggestPeer and Healthy methods for provided hex-encoded addresses.
// Argument pivotAddr is the address of the kademlia.
func testKademliaCase(t *testing.T, pivotAddr string, addrs ...string) {
	addr := common.FromHex(pivotAddr)
	addrs = append(addrs, pivotAddr)

	k := NewKademlia(addr, NewKadParams())

	as := make([][]byte, len(addrs))
	for i, a := range addrs {
		as[i] = common.FromHex(a)
	}

	for _, a := range as {
		if bytes.Equal(a, addr) {
			continue
		}
		p := &BzzAddr{OAddr: a, UAddr: a}
		if err := k.Register([]OverlayAddr{p}); err != nil {
			t.Fatal(err)
		}
	}

	ppmap := NewPeerPotMap(2, as)

	pp := ppmap[pivotAddr]

	for {
		a, _, _ := k.SuggestPeer()
		if a == nil {
			break
		}
		k.On(&BzzPeer{BzzAddr: a.(*BzzAddr)})
	}

	h := k.Healthy(pp)
	if !(h.GotNN && h.KnowNN && h.Full) {
		t.Error("not healthy")
	}
}

/*
The regression test for the following invalid kademlia edge case.

Addresses used in this test are discovered as part of the simulation network
in higher level tests for streaming. They were generated randomly.

=========================================================================
Mon Apr  9 12:18:24 UTC 2018 KΛÐΞMLIΛ hive: queen's address: 7efef1
population: 9 (49), MinProxBinSize: 2, MinBinSize: 2, MaxBinSize: 4
000  2 d7e5 ec56                    | 18 ec56 (0) d7e5 (0) d9e0 (0) c735 (0)
001  2 18f1 3176                    | 14 18f1 (0) 10bb (0) 10d1 (0) 0421 (0)
002  2 52aa 47cd                    | 11 52aa (0) 51d9 (0) 5161 (0) 5130 (0)
003  1 646e                         |  1 646e (0)
004  0                              |  3 769c (0) 76d1 (0) 7656 (0)
============ DEPTH: 5 ==========================================
005  1 7a48                         |  1 7a48 (0)
006  1 7cbd                         |  1 7cbd (0)
007  0                              |  0
008  0                              |  0
009  0                              |  0
010  0                              |  0
011  0                              |  0
012  0                              |  0
013  0                              |  0
014  0                              |  0
015  0                              |  0
=========================================================================
*/
func TestKademliaCase1(t *testing.T) {
	testKademliaCase(t,
		"7efef1c41d77f843ad167be95f6660567eb8a4a59f39240000cce2e0d65baf8e",
		"ec560e6a4806aa37f147ee83687f3cf044d9953e61eedb8c34b6d50d9e2c5623",
		"646e9540c84f6a2f9cf6585d45a4c219573b4fd1b64a3c9a1386fc5cf98c0d4d",
		"18f13c5fba653781019025ab10e8d2fdc916d6448729268afe9e928ffcdbb8e8",
		"317617acf99b4ffddda8a736f8fc6c6ede0bf690bc23d834123823e6d03e2f69",
		"d7e52d9647a5d1c27a68c3ee65d543be3947ae4b68537b236d71ef9cb15fb9ab",
		"7a48f75f8ca60487ae42d6f92b785581b40b91f2da551ae73d5eae46640e02e8",
		"7cbd42350bde8e18ae5b955b5450f8e2cef3419f92fbf5598160c60fd78619f0",
		"52aa3ddec61f4d48dd505a2385403c634f6ad06ee1d99c5c90a5ba6006f9af9c",
		"47cdb6fa93eeb8bc91a417ff4e3b14a9c2ea85137462e2f575fae97f0c4be60d",
		"5161943eb42e2a03e715fe8afa1009ff5200060c870ead6ab103f63f26cb107f",
		"a38eaa1255f76bf883ca0830c86e8c4bb7eed259a8348aae9b03f21f90105bee",
		"b2522bdf1ab26f324e75424fdf6e493b47e8a27687fe76347607b344fc010075",
		"5bd7213964efb2580b91d02ac31ef126838abeba342f5dbdbe8d4d03562671a2",
		"0b531adb82744768b694d7f94f73d4f0c9de591266108daeb8c74066bfc9c9ca",
		"28501f59f70e888d399570145ed884353e017443c675aa12731ada7c87ea14f7",
		"4a45f1fc63e1a9cb9dfa44c98da2f3d20c2923e5d75ff60b2db9d1bdb0c54d51",
		"b193431ee35cd32de95805e7c1c749450c47486595aae7195ea6b6019a64fd61",
		"baebf36a1e35a7ed834e1c72faf44ba16c159fa47d3289ceb3ca35fefa8739b5",
		"a3659bd32e05fa36c8d20dbaaed8362bf1a8a7bd116aed62d8a43a2efbdf513f",
		"10d1b50881a4770ebebdd0a75589dabb931e6716747b0f65fd6b080b88c4fdb6",
		"3c76b8ca5c7ce6a03320646826213f59229626bf5b9d25da0c3ec0662dcb8ff3",
		"4d72a04ddeb851a68cd197ef9a92a3e2ff01fbbff638e64929dd1a9c2e150112",
		"c7353d320987956075b5bc1668571c7a36c800d5598fdc4832ec6569561e15d1",
		"d9e0c7c90878c20ab7639d5954756f54775404b3483407fe1b483635182734f6",
		"8fca67216b7939c0824fb06c5279901a94da41da9482b000f56df9906736ee75",
		"460719d7f7aa7d7438f0eaf30333484fa3bd0f233632c10ba89e6e46dd3604be",
		"0421d92c8a1c79ed5d01305a3d25aaf22a8f5f9e3d4bc80da47ee16ce20465fe",
		"3441d9d9c0f05820a1bb6459fc7d8ef266a1bd929e7db939a10f544efe8261ea",
		"ab198a66c293586746758468c610e5d3914d4ce629147eff6dd55a31f863ff8f",
		"3a1c8c16b0763f3d2c35269f454ff779d1255e954d2deaf6c040fb3f0bcdc945",
		"5561c0ea3b203e173b11e6aa9d0e621a4e10b1d8b178b8fe375220806557b823",
		"7656caccdc79cd8d7ce66d415cc96a718e8271c62fb35746bfc2b49faf3eebf3",
		"5130594fd54c1652cf2debde2c4204573ed76555d1e26757fe345b409af1544a",
		"76d1e83c71ca246d042e37ff1db181f2776265fbcfdc890ce230bfa617c9c2f0",
		"89580231962624c53968c1b0095b4a2732b2a2640a19fdd7d21fd064fcc0a5ef",
		"3d10d001fff44680c7417dd66ecf2e984f0baa20a9bbcea348583ba5ff210c4f",
		"43754e323f0f3a1155b1852bd6edd55da86b8c4cfe3df8b33733fca50fc202b8",
		"a9e7b1bb763ae6452ddcacd174993f82977d81a85206bb2ae3c842e2d8e19b4c",
		"10bb07da7bc7c7757f74149eff167d528a94a253cdc694a863f4d50054c00b6d",
		"28f0bc1b44658548d6e05dd16d4c2fe77f1da5d48b6774bc4263b045725d0c19",
		"835fbbf1d16ba7347b6e2fc552d6e982148d29c624ea20383850df3c810fa8fc",
		"8e236c56a77d7f46e41e80f7092b1a68cd8e92f6156365f41813ad1ca2c6b6f3",
		"51d9c857e9238c49186e37b4eccf17a82de3d5739f026f6043798ab531456e73",
		"bbddf7db6a682225301f36a9fd5b0d0121d2951753e1681295f3465352ad511f",
		"2690a910c33ee37b91eb6c4e0731d1d345e2dc3b46d308503a6e85bbc242c69e",
		"769ce86aa90b518b7ed382f9fdacfbed93574e18dc98fe6c342e4f9f409c2d5a",
		"ba3bebec689ce51d3e12776c45f80d25164fdfb694a8122d908081aaa2e7122c",
		"3a51f4146ea90a815d0d283d1ceb20b928d8b4d45875e892696986a3c0d8fb9b",
		"81968a2d8fb39114342ee1da85254ec51e0608d7f0f6997c2a8354c260a71009",
	)
}

/*
The regression test for the following invalid kademlia edge case.

Addresses used in this test are discovered as part of the simulation network
in higher level tests for streaming. They were generated randomly.

=========================================================================
Mon Apr  9 18:43:48 UTC 2018 KΛÐΞMLIΛ hive: queen's address: bc7f3b
population: 9 (49), MinProxBinSize: 2, MinBinSize: 2, MaxBinSize: 4
000  2 0f49 67ff                    | 28 0f49 (0) 0211 (0) 07b2 (0) 0703 (0)
001  2 e84b f3a4                    | 13 f3a4 (0) e84b (0) e58b (0) e60b (0)
002  1 8dba                         |  1 8dba (0)
003  2 a008 ad72                    |  2 ad72 (0) a008 (0)
004  0                              |  3 b61f (0) b27f (0) b027 (0)
============ DEPTH: 5 ==========================================
005  1 ba19                         |  1 ba19 (0)
006  0                              |  0
007  1 bdd6                         |  1 bdd6 (0)
008  0                              |  0
009  0                              |  0
010  0                              |  0
011  0                              |  0
012  0                              |  0
013  0                              |  0
014  0                              |  0
015  0                              |  0
=========================================================================
*/
func TestKademliaCase2(t *testing.T) {
	testKademliaCase(t,
		"bc7f3b6a4a7e3c91b100ca6680b6c06ff407972b88956324ca853295893e0237", "67ffb61d3aa27449d277016188f35f19e2321fbda5008c68cf6303faa080534f", "600cd54c842eadac1729c04abfc369bc244572ca76117105b9dd910283b82730", "d955a05409650de151218557425105a8aa2867bb6a0e0462fa1cf90abcf87ad6", "7a6b726de45abdf7bb3e5fd9fb0dc8932270ca4dedef92238c80c05bcdb570e3", "263e99424ebfdb652adb4e3dcd27d59e11bb7ae1c057b3ef6f390d0228006254", "ba195d1a53aafde68e661c64d39db8c2a73505bf336125c15c3560de3b48b7ed", "3458c762169937115f67cabc35a6c384ed70293a8aec37b077a6c1b8e02d510e", "4ef4dc2e28ac6efdba57e134ac24dd4e0be68b9d54f7006515eb9509105f700c", "2a8782b79b0c24b9714dfd2c8ff1932bebc08aa6520b4eaeaa59ff781238890c", "625d02e960506f4524e9cdeac85b33faf3ea437fceadbd478b62b78720cf24fc", "e051a36a8c8637f520ba259c9ed3fadaf740dadc6a04c3f0e21778ebd4cd6ac4", "e34bc014fa2504f707bb3d904872b56c2fa250bee3cb19a147a0418541f1bd90", "28036dc79add95799916893890add5d8972f3b95325a509d6ded3d448f4dc652", "1b013c407794fa2e4c955d8f51cbc6bd78588a174b6548246b291281304b5409", "34f71b68698e1534095ff23ee9c35bf64c7f12b8463e7c6f6b19c25cf03928b4", "c712c6e9bbb7076832972a95890e340b94ed735935c3c0bb788e61f011b59479", "a008d5becdcda4b9dbfdaafc3cec586cf61dcf2d4b713b6168fff02e3b9f0b08", "29de15555cdbebaab214009e416ee92f947dcec5dab9894129f50f1b17138f34", "5df9449f700bd4b5a23688b68b293f2e92fa6ca524c93bc6bb9936efba9d9ada", "3ab0168a5f87fedc6a39b53c628256ac87a98670d8691bbdaaecec22418d13a2", "1ee299b2d2a74a568494130e6869e66d57982d345c482a0e0eeb285ac219ae3b", "e0e0e3b860cea9b7a74cf1b0675cc632dc64e80a02f20bbc5e96e2e8bb670606", "dc1ba6f169b0fcdcca021dcebaf39fe5d4875e7e69b854fad65687c1d7719ec0", "d321f73e42fcfb1d3a303eddf018ca5dffdcfd5567cd5ec1212f045f6a07e47d", "070320c3da7b542e5ca8aaf6a0a53d2bb5113ed264ab1db2dceee17c729edcb1", "17d314d65fdd136b50d182d2c8f5edf16e7838c2be8cf2c00abe4b406dbcd1d8", "e60b99e0a06f7d2d99d84085f67cdf8cc22a9ae22c339365d80f90289834a2b4", "02115771e18932e1f67a45f11f5bf743c5dae97fbc477d34d35c996012420eac", "3102a40eb2e5060353dd19bf61eeec8782dd1bebfcb57f4c796912252b591827", "8dbaf231062f2dc7ddaba5f9c7761b0c21292be51bf8c2ef503f31d4a2f63f79", "b02787b713c83a9f9183216310f04251994e04c2763a9024731562e8978e7cc4", "b27fe6cd33989e10909ce794c4b0b88feae286b614a59d49a3444c1a7b51ea82", "07b2d2c94fdc6fd148fe23be2ed9eff54f5e12548f29ed8416e6860fc894466f", "e58bf9f451ef62ac44ff0a9bb0610ec0fd14d423235954f0d3695e83017cbfc4", "bdd600b91bb79d1ee0053b854de308cfaa7e2abce575ea6815a0a7b3449609c2", "0f49c93c1edc7999920b21977cedd51a763940dac32e319feb9c1df2da0f3071", "7cbf0297cd41acf655cd6f960d7aaf61479edb4189d5c001cbc730861f0deb41", "79265193778d87ad626a5f59397bc075872d7302a12634ce2451a767d0a82da2", "2fe7d705f7c370b9243dbaafe007d555ff58d218822fca49d347b12a0282457c", "e84bc0c83d05e55a0080eed41dda5a795da4b9313a4da697142e69a65834cbb3", "cc4d278bd9aa0e9fb3cd8d2e0d68fb791aab5de4b120b845c409effbed47a180", "1a2317a8646cd4b6d3c4aa4cc25f676533abb689cf180787db216880a1239ad8", "cbafd6568cf8e99076208e6b6843f5808a7087897c67aad0c54694669398f889", "7b7c8357255fc37b4dae0e1af61589035fd39ff627e0938c6b3da8b4e4ec5d23", "2b8d782c1f5bac46c922cf439f6aa79f91e9ba5ffc0020d58455188a2075b334", "b61f45af2306705740742e76197a119235584ced01ef3f7cf3d4370f6c557cd1", "2775612e7cdae2780bf494c370bdcbe69c55e4a1363b1dc79ea0135e61221cce", "f3a49bb22f40885e961299abfa697a7df690a79f067bf3a4847a3ad48d826c9f", "ad724ac218dc133c0aadf4618eae21fdd0c2f3787af279846b49e2b4f97ff167",
	)
}

/*
The regression test for the following invalid kademlia edge case.

Addresses used in this test are discovered as part of the simulation network
in higher level tests for streaming. They were generated randomly.

=========================================================================
Mon Apr  9 19:04:35 UTC 2018 KΛÐΞMLIΛ hive: queen's address: b4822e
population: 8 (49), MinProxBinSize: 2, MinBinSize: 2, MaxBinSize: 4
000  2 786c 774b                    | 29 774b (0) 786c (0) 7a79 (0) 7d2f (0)
001  2 d9de cf19                    | 10 cf19 (0) d9de (0) d2ff (0) d2a2 (0)
002  2 8ca1 8d74                    |  5 8d74 (0) 8ca1 (0) 9793 (0) 9f51 (0)
003  0                              |  0
004  0                              |  3 bfac (0) bcbb (0) bde9 (0)
005  0                              |  0
============ DEPTH: 6 ==========================================
006  1 b660                         |  1 b660 (0)
007  0                              |  0
008  1 b450                         |  1 b450 (0)
009  0                              |  0
010  0                              |  0
011  0                              |  0
012  0                              |  0
013  0                              |  0
014  0                              |  0
015  0                              |  0
=========================================================================
*/
func TestKademliaCase3(t *testing.T) {
	testKademliaCase(t,
		"b4822e874a01b94ac3a35c821e6db131e785c2fcbb3556e84b36102caf09b091", "2ecf54ea38d58f9cfc3862e54e5854a7c506fbc640e0b38e46d7d45a19794999", "442374092be50fc7392e8dd3f6fab3158ff7f14f26ff98060aed9b2eecf0b97d", "b450a4a67fcfa3b976cf023d8f1f15052b727f712198ce901630efe2f95db191", "9a7291638eb1c989a6dd6661a42c735b23ac6605b5d3e428aa5ffe650e892c85", "67f62eeab9804cfcac02b25ebeab9113d1b9d03dd5200b1c5a324cc0163e722f", "2e4a0e4b53bca4a9d7e2734150e9f579f29a255ade18a268461b20d026c9ee90", "30dd79c5fcdaa1b106f6960c45c9fde7c046aa3d931088d98c52ab759d0b2ac4", "97936fb5a581e59753c54fa5feec493714f2218245f61f97a62eafd4699433e4", "3a2899b6e129e3e193f6e2aefb82589c948c246d2ec1d4272af32ef3b2660f44", "f0e2a8aa88e67269e9952431ef12e5b29b7f41a1871fbfc38567fad95655d607", "7fa12b3f3c5f8383bfc644b958f72a486969733fa097d8952b3eb4f7b4f73192", "360c167aad5fc992656d6010ec45fdce5bcd492ad9608bc515e2be70d4e430c1", "fe21bc969b3d8e5a64a6484a829c1e04208f26f3cd4de6afcbc172a5bd17f1f1", "b660a1f40141d7ccd282fe5bd9838744119bd1cb3780498b5173578cc5ad308f", "44dcb3370e76680e2fba8cd986ad45ff0b77ca45680ee8d950e47922c4af6226", "8ca126923d17fccb689647307b89f38aa14e2a7b9ebcf3c1e31ccf3d2291a3bc", "f0ae19ae9ce6329327cbf42baf090e084c196b0877d8c7b69997e0123be23ef8", "d2a2a217385158e3e1e348883a14bc423e57daa12077e8c49797d16121ea0810", "f5467ccd85bb4ebe768527db520a210459969a5f1fae6e07b43f519799f0b224", "68be5fd9f9d142a5099e3609011fe3bab7bb992c595999e31e0b3d1668dfb3cf", "4d49a8a476e4934afc6b5c36db9bece3ed1804f20b952da5a21b2b0de766aa73", "ea7155745ef3fb2d099513887a2ba279333ced65c65facbd890ce58bd3fce772", "cf19f51f4e848053d289ac95a9138cdd23fc3077ae913cd58cda8cc7a521b2e1", "590b1cd41c7e6144e76b5cd515a3a4d0a4317624620a3f1685f43ae68bdcd890", "d2ffe0626b5f94a7e00fa0b506e7455a3d9399c15800db108d5e715ef5f6e346", "69630878c50a91f6c2edd23a706bfa0b50bd5661672a37d67bab38e6bca3b698", "445e9067079899bb5faafaca915ae6c0f6b1b730a5a628835dd827636f7feb1e", "6461c77491f1c4825958949f23c153e6e1759a5be53abbcee17c9da3867f3141", "23a235f4083771ccc207771daceda700b525a59ab586788d4f6892e69e34a6e2", "bde99f79ef41a81607ddcf92b9f95dcbc6c3537e91e8bf740e193dc73b19485e", "177957c0e5f0fbd12b88022a91768095d193830986caec8d888097d3ff4310b8", "bcbbdbaa4cdf8352422072f332e05111b732354a35c4d7c617ce1fc3b8b42a5a", "774b6717fdfb0d1629fb9d4c04a9ca40079ae2955d7f82e897477055ed017abb", "16443bf625be6d39ecaa6f114e5d2c1d47a64bfd3c13808d94b55b6b6acef2ee", "8d7495d9008066505ed00ce8198af82bfa5a6b4c08768b4c9fb3aa4eb0b0cca2", "15800849a53349508cb382959527f6c3cf1a46158ff1e6e2316b7dea7967e35f", "7a792f0f4a2b731781d1b244b2a57947f1a2e32900a1c0793449f9f7ae18a7b7", "5e517c2832c9deaa7df77c7bad4d20fd6eda2b7815e155e68bc48238fac1416f", "9f51a14f0019c72bd1d472706d8c80a18c1873c6a0663e754b60eae8094483d7", "7d2fabb565122521d22ba99fed9e5be6a458fbc93156d54db27d97a00b8c3a97", "786c9e412a7db4ec278891fa534caa9a1d1a028c631c6f3aeb9c4d96ad895c36", "3bd6341d40641c2632a5a0cd7a63553a04e251efd7195897a1d27e02a7a8bfde", "31efd1f5fb57b8cff0318d77a1a9e8d67e1d1c8d18ce90f99c3a240dff48cdc8", "d9de3e1156ce1380150948acbcfecd99c96e7f4b0bc97745f4681593d017f74f", "427a2201e09f9583cd990c03b81b58148c297d474a3b50f498d83b1c7a9414cd", "bfaca11596d3dec406a9fcf5d97536516dfe7f0e3b12078428a7e1700e25218a", "351c4770a097248a650008152d0cab5825d048bef770da7f3364f59d1e721bc0", "ee00f205d1486b2be7381d962bd2867263758e880529e4e2bfedfa613bbc0e71", "6aa3b6418d89e3348e4859c823ef4d6d7cd46aa7f7e77aba586c4214d760d8f8",
	)
}

/*
The regression test for the following invalid kademlia edge case.

Addresses used in this test are discovered as part of the simulation network
in higher level tests for streaming. They were generated randomly.

=========================================================================
Mon Apr  9 19:16:25 UTC 2018 KΛÐΞMLIΛ hive: queen's address: 9a90fe
population: 8 (49), MinProxBinSize: 2, MinBinSize: 2, MaxBinSize: 4
000  2 72ef 4e6c                    | 24 0b1e (0) 0d66 (0) 17f5 (0) 17e8 (0)
001  2 fc2b fa47                    | 13 fa47 (0) fc2b (0) fffd (0) ecef (0)
002  2 b847 afa8                    |  6 afa8 (0) ad77 (0) bb7c (0) b847 (0)
003  0                              |  0
004  0                              |  4 91fc (0) 957d (0) 9482 (0) 949a (0)
============ DEPTH: 5 ==========================================
005  1 9ccf                         |  1 9ccf (0)
006  0                              |  0
007  1 9bb2                         |  1 9bb2 (0)
008  0                              |  0
009  0                              |  0
010  0                              |  0
011  0                              |  0
012  0                              |  0
013  0                              |  0
014  0                              |  0
015  0                              |  0
=========================================================================
*/
func TestKademliaCase4(t *testing.T) {
	testKademliaCase(t,
		"9a90fe3506277244549064b8c3276abb06284a199d9063a97331947f2b7da7f4",
		"c19359eddef24b7be1a833b4475f212cd944263627a53f9ef4837d106c247730", "fc2b6fef99ef947f7e57c3df376891769e2a2fd83d2b8e634e0fc1e91eaa080c", "ecefc0e1a8ea7bb4b48c469e077401fce175dd75294255b96c4e54f6a2950a55", "bb7ce598efc056bba343cc2614aa3f67a575557561290b44c73a63f8f433f9f7", "55fbee6ca52dfd7f0be0db969ee8e524b654ab4f0cce7c05d83887d7d2a15460", "afa852b6b319998c6a283cc0c82d2f5b8e9410075d7700f3012761f1cfbd0f76", "36c370cfb63f2087971ba6e58d7585b04e16b8f0da335efb91554c2dd8fe191c", "6be41e029985edebc901fb77fc4fb65516b6d85086e2a98bfa3159c99391e585", "dd3cfc72ea553e7d2b28f0037a65646b30955b929d29ba4c40f4a2a811248e77", "da3a8f18e09c7b0ca235c4e33e1441a5188f1df023138bf207753ee63e768f7d", "de9e3ab4dc572d54a2d4b878329fd832bb51a149f4ce167316eeb177b61e7e01", "4e6c1ecde6ed917706257fe020a1d02d2e9d87fca4c85f0f7b132491008c5032", "72ef04b77a070e13463b3529dd312bcacfb7a12d20dc597f5ec3de0501e9b834", "3fef57186675d524ab8bb1f54ba8cb68610babca1247c0c46dbb60aed003c69d", "1d8e6b71f7a052865d6558d4ba44ad5fab7b908cc1badf5766822e1c20d0d823", "6be2f2b4ffa173014d4ec7df157d289744a2bda54bb876b264ccfa898a0da315", "b0ba3fff8643f9985c744327b0c4c869763509fd5da2de9a80a4a0a082021255", "9ccf40b9406ba2e6567101fb9b4e5334a9ec74263eff47267da266ba45e6c158", "d7347f02c180a448e60f73931845062ce00048750b584790278e9c93ef31ad81", "b68c6359a22b3bee6fecb8804311cfd816648ea31d530c9fb48e477e029d707a", "0d668a18ad7c2820214df6df95a6c855ce19fb1cb765f8ca620e45db76686d37", "3fbd2663bff65533246f1fabb9f38086854c6218aeb3dc9ac6ac73d4f0988f91", "949aa5719ca846052bfaa1b38c97b6eca3df3e24c0e0630042c6bccafbb4cdb5", "77b8a2b917bef5d54f3792183b014cca7798f713ff14fe0b2ac79b4c9f6f996d", "17e853cbd8dc00cba3cd9ffeb36f26a9f41a0eb92f80b62c2cda16771c935388", "5f682ed7a8cf2f98387c3def7c97f9f05ae39e39d393eeca3cf621268d6347f8", "ad77487eaf11fd8084ba4517a51766eb0e5b77dd3492dfa79aa3a2802fb29d20", "d247cfcacf9a8200ebaddf639f8c926ab0a001abe682f40df3785e80ed124e91", "195589442e11907eede1ee6524157f1125f68399f3170c835ff81c603b069f6c", "5b5ca0a67f3c54e7d3a6a862ef56168ec9ed1f4945e6c24de6d336b2be2e6f8c", "56430e4caa253015f1f998dce4a48a88af1953f68e94eca14f53074ae9c3e467", "0b1eed6a5bf612d1d8e08f5c546f3d12e838568fd3aa43ed4c537f10c65545d6", "7058db19a56dfff01988ac4a62e1310597f9c8d7ebde6890dadabf047d722d39", "b847380d6888ff7cd11402d086b19eccc40950b52c9d67e73cb4f8462f5df078", "df6c048419a2290ab546d527e9eeba349e7f7e1759bafe4adac507ce60ef9670", "91fc5b4b24fc3fbfea7f9a3d0f0437cb5733c0c2345d8bdffd7048d6e3b8a37b", "957d8ea51b37523952b6f5ae95462fcd4aed1483ef32cc80b69580aaeee03606", "efa82e4e91ad9ab781977400e9ac0bb9de7389aaedebdae979b73d1d3b8d72b0", "7400c9f3f3fc0cc6fe8cc37ab24b9771f44e9f78be913f73cd35fc4be030d6bd", "9bb28f4122d61f7bb56fe27ef706159fb802fef0f5de9dfa32c9c5b3183235f1", "40a8de6e98953498b806614532ea4abf8b99ad7f9719fb68203a6eae2efa5b2a", "412de0b218b8f7dcacc9205cd16ffb4eca5b838f46a2f4f9f534026061a47308", "17f56ecad51075080680ad9faa0fd8946b824d3296ddb20be07f9809fe8d1c5a", "fffd4e7ae885a41948a342b6647955a7ec8a8039039f510cff467ef597675457", "35e78e11b5ac46a29dd04ab0043136c3291f4ca56cb949ace33111ed56395463", "94824fc80230af82077c83bfc01dc9675b1f9d3d538b1e5f41c21ac753598691", "fa470ae314ca3fce493f21b423eef2a49522e09126f6f2326fa3c9cac0b344f7", "7078860b5b621b21ac7b95f9fc4739c8235ce5066a8b9bd7d938146a34fa88ec", "eea53560f0428bfd2eca4f86a5ce9dec5ff1309129a975d73465c1c9e9da71d1",
	)
}

/*
The regression test for the following invalid kademlia edge case.

Addresses used in this test are discovered as part of the simulation network
in higher level tests for streaming. They were generated randomly.

=========================================================================
Mon Apr  9 19:25:18 UTC 2018 KΛÐΞMLIΛ hive: queen's address: 5dd5c7
population: 13 (49), MinProxBinSize: 2, MinBinSize: 2, MaxBinSize: 4
000  2 e528 fad0                    | 22 fad0 (0) e528 (0) e3bb (0) ed13 (0)
001  3 3f30 18e0 1dd3               |  7 3f30 (0) 23db (0) 10b6 (0) 18e0 (0)
002  4 7c54 7804 61e4 60f9          | 10 61e4 (0) 60f9 (0) 636c (0) 7186 (0)
003  2 40ae 4bae                    |  5 4bae (0) 4d5c (0) 403a (0) 40ae (0)
004  0                              |  0
005  0                              |  3 5808 (0) 5a0e (0) 5bdb (0)
============ DEPTH: 6 ==========================================
006  2 5f14 5f61                    |  2 5f14 (0) 5f61 (0)
007  0                              |  0
008  0                              |  0
009  0                              |  0
010  0                              |  0
011  0                              |  0
012  0                              |  0
013  0                              |  0
014  0                              |  0
015  0                              |  0
=========================================================================
*/
func TestKademliaCase5(t *testing.T) {
	testKademliaCase(t,
		"5dd5c77dd9006a800478fcebb02d48d4036389e7d3c8f6a83b97dbad13f4c0a9",
		"78fafa0809929a1279ece089a51d12457c2d8416dff859aeb2ccc24bb50df5ec", "1dd39b1257e745f147cbbc3cadd609ccd6207c41056dbc4254bba5d2527d3ee5", "5f61dd66d4d94aec8fcc3ce0e7885c7edf30c43143fa730e2841c5d28e3cd081", "8aa8b0472cb351d967e575ad05c4b9f393e76c4b01ef4b3a54aac5283b78abc9", "4502f385152a915b438a6726ce3ea9342e7a6db91a23c2f6bee83a885ed7eb82", "718677a504249db47525e959ef1784bed167e1c46f1e0275b9c7b588e28a3758", "7c54c6ed1f8376323896ed3a4e048866410de189e9599dd89bf312ca4adb96b5", "18e03bd3378126c09e799a497150da5c24c895aedc84b6f0dbae41fc4bac081a", "23db76ac9e6e58d9f5395ca78252513a7b4118b4155f8462d3d5eec62486cadc", "40ae0e8f065e96c7adb7fa39505136401f01780481e678d718b7f6dbb2c906ec", "c1539998b8bae19d339d6bbb691f4e9daeb0e86847545229e80fe0dffe716e92", "ed139d73a2699e205574c08722ca9f030ad2d866c662f1112a276b91421c3cb9", "5bdb19584b7a36d09ca689422ef7e6bb681b8f2558a6b2177a8f7c812f631022", "636c9de7fe234ffc15d67a504c69702c719f626c17461d3f2918e924cd9d69e2", "de4455413ff9335c440d52458c6544191bd58a16d85f700c1de53b62773064ea", "de1963310849527acabc7885b6e345a56406a8f23e35e436b6d9725e69a79a83", "a80a50a467f561210a114cba6c7fb1489ed43a14d61a9edd70e2eb15c31f074d", "7804f12b8d8e6e4b375b242058242068a3809385e05df0e64973cde805cf729c", "60f9aa320c02c6f2e6370aa740cf7cea38083fa95fca8c99552cda52935c1520", "d8da963602390f6c002c00ce62a84b514edfce9ebde035b277a957264bb54d21", "8463d93256e026fe436abad44697152b9a56ac8e06a0583d318e9571b83d073c", "9a3f78fcefb9a05e40a23de55f6153d7a8b9d973ede43a380bf46bb3b3847de1", "e3bb576f4b3760b9ca6bff59326f4ebfc4a669d263fb7d67ab9797adea54ed13", "4d5cdbd6dcca5bdf819a0fe8d175dc55cc96f088d37462acd5ea14bc6296bdbe", "5a0ed28de7b5258c727cb85447071c74c00a5fbba9e6bc0393bc51944d04ab2a", "61e4ddb479c283c638f4edec24353b6cc7a3a13b930824aad016b0996ca93c47", "7e3610868acf714836cafaaa7b8c009a9ac6e3a6d443e5586cf661530a204ee2", "d74b244d4345d2c86e30a097105e4fb133d53c578320285132a952cdaa64416e", "cfeed57d0f935bfab89e3f630a7c97e0b1605f0724d85a008bbfb92cb47863a8", "580837af95055670e20d494978f60c7f1458dc4b9e389fc7aa4982b2aca3bce3", "df55c0c49e6c8a83d82dfa1c307d3bf6a20e18721c80d8ec4f1f68dc0a137ced", "5f149c51ce581ba32a285439a806c063ced01ccd4211cd024e6a615b8f216f95", "1eb76b00aeb127b10dd1b7cd4c3edeb4d812b5a658f0feb13e85c4d2b7c6fe06", "7a56ba7c3fb7cbfb5561a46a75d95d7722096b45771ec16e6fa7bbfab0b35dfe", "4bae85ad88c28470f0015246d530adc0cd1778bdd5145c3c6b538ee50c4e04bd", "afd1892e2a7145c99ec0ebe9ded0d3fec21089b277a68d47f45961ec5e39e7e0", "953138885d7b36b0ef79e46030f8e61fd7037fbe5ce9e0a94d728e8c8d7eab86", "de761613ef305e4f628cb6bf97d7b7dc69a9d513dc233630792de97bcda777a6", "3f3087280063d09504c084bbf7fdf984347a72b50d097fd5b086ffabb5b3fb4c", "7d18a94bb1ebfdef4d3e454d2db8cb772f30ca57920dd1e402184a9e598581a0", "a7d6fbdc9126d9f10d10617f49fb9f5474ffe1b229f76b7dd27cebba30eccb5d", "fad0246303618353d1387ec10c09ee991eb6180697ed3470ed9a6b377695203d", "1cf66e09ea51ee5c23df26615a9e7420be2ac8063f28f60a3bc86020e94fe6f3", "8269cdaa153da7c358b0b940791af74d7c651cd4d3f5ed13acfe6d0f2c539e7f", "90d52eaaa60e74bf1c79106113f2599471a902d7b1c39ac1f55b20604f453c09", "9788fd0c09190a3f3d0541f68073a2f44c2fcc45bb97558a7c319f36c25a75b3", "10b68fc44157ecfdae238ee6c1ce0333f906ad04d1a4cb1505c8e35c3c87fbb0", "e5284117fdf3757920475c786e0004cb00ba0932163659a89b36651a01e57394", "403ad51d911e113dcd5f9ff58c94f6d278886a2a4da64c3ceca2083282c92de3",
	)
}
