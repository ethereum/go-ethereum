package pss

import (
	"bytes"
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/pot"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/storage"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
)

const (
	pssServiceName = "pss"
	bzzServiceName = "bzz"
)

var (
	snapshotfile string
	debugflag    = flag.Bool("v", false, "verbose")

	w    *whisper.Whisper
	wapi *whisper.PublicWhisperAPI

	// custom logging
	psslogmain log.Logger
)

var services = newServices()

func init() {

	flag.Parse()
	rand.Seed(time.Now().Unix())

	adapters.RegisterServices(services)

	loglevel := log.LvlDebug
	if *debugflag {
		loglevel = log.LvlTrace
	}

	psslogmain = log.New("psslog", "*")
	hs := log.StreamHandler(os.Stderr, log.TerminalFormat(true))
	hf := log.LvlFilterHandler(loglevel, hs)
	h := log.CallerFileHandler(hf)
	log.Root().SetHandler(h)

	w = whisper.New()
	wapi = whisper.NewPublicWhisperAPI(w)
}

func TestAddressMatch(t *testing.T) {

	localaddr := network.RandomAddr().Over()
	copy(localaddr[:8], []byte("deadbeef"))
	remoteaddr := []byte("feedbeef")
	kadparams := network.NewKadParams()
	kad := network.NewKademlia(localaddr, kadparams)
	keys, err := wapi.NewKeyPair()
	if err != nil {
		t.Fatalf("Could not generate private key: %v", err)
	}
	privkey, err := w.GetPrivateKey(keys)
	pssp := NewPssParams(privkey)
	ps := NewPss(kad, nil, pssp)

	pssmsg := &PssMsg{
		To:      remoteaddr,
		Payload: &whisper.Envelope{},
	}

	// differ from first byte
	if ps.isSelfRecipient(pssmsg) {
		t.Fatalf("isSelfRecipient true but %x != %x", remoteaddr, localaddr)
	}
	if ps.isSelfPossibleRecipient(pssmsg) {
		t.Fatalf("isSelfPossibleRecipient true but %x != %x", remoteaddr[:8], localaddr[:8])
	}

	// 8 first bytes same
	copy(remoteaddr[:4], localaddr[:4])
	if ps.isSelfRecipient(pssmsg) {
		t.Fatalf("isSelfRecipient true but %x != %x", remoteaddr, localaddr)
	}
	if !ps.isSelfPossibleRecipient(pssmsg) {
		t.Fatalf("isSelfPossibleRecipient false but %x == %x", remoteaddr[:8], localaddr[:8])
	}

	// all bytes same
	pssmsg.To = localaddr
	if !ps.isSelfRecipient(pssmsg) {
		t.Fatalf("isSelfRecipient false but %x == %x", remoteaddr, localaddr)
	}
	if !ps.isSelfPossibleRecipient(pssmsg) {
		t.Fatalf("isSelfPossibleRecipient false but %x == %x", remoteaddr[:8], localaddr[:8])
	}
}

// tests:
// sets public key for peer
// set an outgoing symmetric key for peer
// generate own symmetric key for incoming message from peer
func TestKeys(t *testing.T) {
	outkey := make([]byte, 32) // whisper aes symkeys are 32, if that changes this test will break

	// make our key and init pss with it
	ourkeys, err := wapi.NewKeyPair()
	if err != nil {
		t.Fatalf("create 'our' key fail")
	}
	theirkeys, err := wapi.NewKeyPair()
	if err != nil {
		t.Fatalf("create 'their' key fail")
	}
	ourprivkey, err := w.GetPrivateKey(ourkeys)
	if err != nil {
		t.Fatalf("failed to retrieve 'our' private key")
	}
	theirprivkey, err := w.GetPrivateKey(theirkeys)
	if err != nil {
		t.Fatalf("failed to retrieve 'their' private key")
	}
	ps := NewTestPss(ourprivkey)

	// set up peer with mock address, mapped to mocked publicaddress and with mocked symkey
	addr := network.RandomAddr().Over()
	topic := whisper.BytesToTopic([]byte("foo:42"))
	ps.SetOutgoingPublicKey(pot.NewAddressFromBytes(addr), topic, &theirprivkey.PublicKey)
	copy(outkey[:16], addr[16:])
	copy(outkey[16:], addr[:16])
	outkeyid, err := ps.SetOutgoingSymmetricKey(pot.NewAddressFromBytes(addr), topic, outkey)
	if err != nil {
		t.Fatalf("failed to set 'our' outgoing symmetric key")
	}

	// make a symmetric key that we will send to peer for encrypting messages to us
	inkeyid, err := ps.GenerateIncomingSymmetricKey(pot.NewAddressFromBytes(addr), topic)
	if err != nil {
		t.Fatalf("failed to set 'our' incoming symmetric key")
	}

	// get the key back from whisper, check that it's still the same
	outkeyback, err := ps.w.GetSymKey(outkeyid)
	if err != nil {
		t.Fatalf(err.Error())
	}
	inkey, err := ps.w.GetSymKey(inkeyid)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if !bytes.Equal(outkeyback, outkey) {
		t.Fatalf("passed outgoing symkey doesnt equal stored: %x / %x", outkey, outkeyback)
	}

	t.Logf("symout: %v", outkeyback)
	t.Logf("symin: %v", inkey)

	// check that the key is stored in the peerpool
	var potaddr pot.Address
	copy(potaddr[:], addr)
	psp := ps.peerPool[potaddr][topic]
	t.Logf("peer:\nrw: %v\npubkey: %v\nrecvsymkey: %v\nsendsymkey: %v\nsymkeyexp: %v", psp.rw, psp.pubkey, psp.recvsymkey, psp.sendsymkey, psp.symkeyexpires)
}

func TestKeysExchange(t *testing.T) {

	// set up two nodes directly connected
	// (we are not testing pss routing here)
	topic := whisper.BytesToTopic([]byte("foo:42"))
	adapter := adapters.NewSimAdapter(services)
	net := simulations.NewNetwork(adapter, &simulations.NetworkConfig{
		ID:             "0",
		DefaultService: "bzz",
	})
	lnode, err := net.NewNodeWithConfig(&adapters.NodeConfig{
		Services: []string{"bzz", "pss"},
	})
	if err != nil {
		t.Fatalf("error creating node 1: %v", err)
	}
	rnode, err := net.NewNodeWithConfig(&adapters.NodeConfig{
		Services: []string{"bzz", "pss"},
	})
	if err != nil {
		t.Fatalf("error creating node 2: %v", err)
	}
	err = net.Start(lnode.ID())
	if err != nil {
		t.Fatalf("error starting node 1: %v", err)
	}
	err = net.Start(rnode.ID())
	if err != nil {
		t.Fatalf("error starting node 2: %v", err)
	}
	err = net.Connect(lnode.ID(), rnode.ID())
	if err != nil {
		t.Fatalf("error connecting nodes: %v", err)
	}
	lclient, err := lnode.Client()
	if err != nil {
		t.Fatalf("create node 1 rpc client fail: %v", err)
	}
	rclient, err := rnode.Client()
	if err != nil {
		t.Fatalf("create node 2 rpc client fail: %v", err)
	}
	loaddr := make([]byte, 32)
	err = lclient.Call(&loaddr, "pss_baseAddr")
	if err != nil {
		t.Fatalf("rpc get node 1 baseaddr fail: %v", err)
	}
	roaddr := make([]byte, 32)
	err = rclient.Call(&roaddr, "pss_baseAddr")
	if err != nil {
		t.Fatalf("rpc get node 2 baseaddr fail: %v", err)
	}

	// retrieve public key from pss instance
	// set this public key reciprocally
	lpubkey := make([]byte, 32)
	err = lclient.Call(&lpubkey, "pss_getPublicKey")
	if err != nil {
		t.Fatalf("rpc get node 1 pubkey fail: %v", err)
	}
	rpubkey := make([]byte, 32)
	err = rclient.Call(&rpubkey, "pss_getPublicKey")
	if err != nil {
		t.Fatalf("rpc get node 2 pubkey fail: %v", err)
	}

	time.Sleep(time.Second) // replace with hive healthy code

	hextopic := fmt.Sprintf("%x", topic)
	lmsgC := make(chan APIMsg)
	lctx, _ := context.WithTimeout(context.Background(), time.Second*10)
	lsub, err := lclient.Subscribe(lctx, "pss", lmsgC, "receive", hextopic)
	log.Trace("lsub", "id", lsub)

	rmsgC := make(chan APIMsg)
	rctx, _ := context.WithTimeout(context.Background(), time.Second*10)
	rsub, err := rclient.Subscribe(rctx, "pss", rmsgC, "receive", hextopic)
	log.Trace("rsub", "id", rsub)

	err = lclient.Call(nil, "pss_setPublicKey", roaddr, hextopic, rpubkey)
	err = rclient.Call(nil, "pss_setPublicKey", loaddr, hextopic, lpubkey)

	// use api test method for generating and sending incoming symkey
	// the peer should save it, then generate and send back its own
	var symkeyid string
	err = lclient.Call(&symkeyid, "psstest_sendSymKey", roaddr, hextopic)
	t.Logf("lclient sym id: %s", symkeyid)
	time.Sleep(time.Second * 2) // replace with sim expect logic

	// after the exchange, the key for receiving on node 1
	// should be the same as sending on node 2, and vice versa
	tmpbytes := make([]byte, defaultSymKeyLength*2)
	lrecvkey := make([]byte, defaultSymKeyLength)
	lsendkey := make([]byte, defaultSymKeyLength)
	err = lclient.Call(&tmpbytes, "psstest_getSymKeys", roaddr, hextopic)
	copy(lrecvkey, tmpbytes[:defaultSymKeyLength])
	copy(lsendkey, tmpbytes[defaultSymKeyLength:])
	rrecvkey := make([]byte, defaultSymKeyLength)
	rsendkey := make([]byte, defaultSymKeyLength)
	err = rclient.Call(&tmpbytes, "psstest_getSymKeys", loaddr, hextopic)
	copy(rrecvkey, tmpbytes[:defaultSymKeyLength])
	copy(rsendkey, tmpbytes[defaultSymKeyLength:])
	if !bytes.Equal(rrecvkey, lsendkey) {
		t.Fatalf("node 2 receive symkey does not match node 1 send symkey: %x != %x", rrecvkey, lsendkey)
	}
	if !bytes.Equal(lrecvkey, rsendkey) {
		t.Fatalf("node 2 send symkey does not match node 1 receive symkey: %x != %x", rsendkey, lrecvkey)
	}

	// at this point we've verified that symkeys are saved and match on each peer
	// now try sending symmetrically encrypted message
	apimsg := APIMsg{
		Msg: []byte("xyzzy"),
		//Addr: roaddr,
		Addr: roaddr,
	}
	err = lclient.Call(nil, "pss_send", hextopic, apimsg)
	select {
	case recvmsg := <-rmsgC:
		if !bytes.Equal(recvmsg.Msg, apimsg.Msg) {
			t.Fatalf("node 2 received payload mismatch: expected %v, got %v", apimsg.Msg, recvmsg.Msg)
		}
	case cerr := <-rctx.Done():
		t.Fatalf("test message timed out: %v", cerr)
	}
	apimsg.Msg = []byte("plugh")
	apimsg.Addr = loaddr
	err = rclient.Call(nil, "pss_send", hextopic, apimsg)
	select {
	case recvmsg := <-lmsgC:
		if !bytes.Equal(recvmsg.Msg, apimsg.Msg) {
			t.Fatalf("node 1 received payload mismatch: expected %v, got %v", apimsg.Msg, recvmsg.Msg)
		}
	case cerr := <-lctx.Done():
		t.Fatalf("test message timed out: %v", cerr)
	}
	lsub.Unsubscribe()
	rsub.Unsubscribe()
}

func TestCache(t *testing.T) {
	var err error
	var potaddr pot.Address
	to, _ := hex.DecodeString("08090a0b0c0d0e0f1011121314150001020304050607161718191a1b1c1d1e1f")
	keys, err := wapi.NewKeyPair()
	privkey, err := w.GetPrivateKey(keys)

	ps := NewTestPss(privkey)
	pp := NewPssParams(privkey)
	data := []byte("foo")
	datatwo := []byte("bar")
	fwdaddr := network.RandomAddr()
	copy(potaddr[:], fwdaddr.Over())
	wparams := &whisper.MessageParams{
		TTL:      DefaultTTL,
		Src:      privkey,
		Dst:      &privkey.PublicKey,
		Topic:    PingTopic,
		WorkTime: defaultWhisperWorkTime,
		PoW:      defaultWhisperPoW,
		Payload:  data,
	}
	woutmsg, err := whisper.NewSentMessage(wparams)
	env, err := woutmsg.Wrap(wparams)
	msg := &PssMsg{
		Payload: env,
		To:      to,
	}
	wparams.Payload = datatwo
	woutmsg, err = whisper.NewSentMessage(wparams)
	envtwo, err := woutmsg.Wrap(wparams)
	msgtwo := &PssMsg{
		Payload: envtwo,
		To:      to,
	}

	digest, err := ps.storeMsg(msg)
	if err != nil {
		t.Fatalf("could not store cache msgone: %v", err)
	}
	digesttwo, err := ps.storeMsg(msgtwo)
	if err != nil {
		t.Fatalf("could not store cache msgtwo: %v", err)
	}

	if digest == digesttwo {
		t.Fatalf("different msgs return same crc: %d", digesttwo)
	}

	// check the cache
	err = ps.addFwdCache(digest)
	if err != nil {
		t.Fatalf("write to pss expire cache failed: %v", err)
	}

	if !ps.checkFwdCache(nil, digest) {
		t.Fatalf("message %v should have EXPIRE record in cache but checkCache returned false", msg)
	}

	if ps.checkFwdCache(nil, digesttwo) {
		t.Fatalf("message %v should NOT have EXPIRE record in cache but checkCache returned true", msgtwo)
	}

	time.Sleep(pp.Cachettl)
	if ps.checkFwdCache(nil, digest) {
		t.Fatalf("message %v should have expired from cache but checkCache returned true", msg)
	}
}

func newServices() adapters.Services {
	stateStore := adapters.NewSimStateStore()
	kademlias := make(map[discover.NodeID]*network.Kademlia)
	kademlia := func(id discover.NodeID) *network.Kademlia {
		if k, ok := kademlias[id]; ok {
			return k
		}
		addr := network.NewAddrFromNodeID(id)
		params := network.NewKadParams()
		params.MinProxBinSize = 2
		params.MaxBinSize = 3
		params.MinBinSize = 1
		params.MaxRetries = 1000
		params.RetryExponent = 2
		params.RetryInterval = 1000000
		kademlias[id] = network.NewKademlia(addr.Over(), params)
		return kademlias[id]
	}
	return adapters.Services{
		//"pss": func(id discover.NodeID, snapshot []byte) node.Service {
		"pss": func(ctx *adapters.ServiceContext) (node.Service, error) {
			cachedir, err := ioutil.TempDir("", "pss-cache")
			if err != nil {
				return nil, fmt.Errorf("create pss cache tmpdir failed", "error", err)
			}
			dpa, err := storage.NewLocalDPA(cachedir)
			if err != nil {
				return nil, fmt.Errorf("local dpa creation failed", "error", err)
			}

			keys, err := wapi.NewKeyPair()
			privkey, err := w.GetPrivateKey(keys)
			pssp := NewPssParams(privkey)
			pskad := kademlia(ctx.Config.ID)
			ps := NewPss(pskad, dpa, pssp)

			ping := &Ping{
				C: make(chan struct{}),
			}
			ps.Register(&PingTopic, RegisterPssProtocol(ps, &PingTopic, PingProtocol, NewPingProtocol(ping.PingHandler)).Handle)
			if err != nil {
				log.Error("Couldnt register pss protocol", "err", err)
				os.Exit(1)
			}

			return ps, nil
		},
		//"bzz": func(id discover.NodeID, snapshot []byte) node.Service {
		"bzz": func(ctx *adapters.ServiceContext) (node.Service, error) {
			addr := network.NewAddrFromNodeID(ctx.Config.ID)
			hp := network.NewHiveParams()
			hp.Discovery = false
			config := &network.BzzConfig{
				OverlayAddr:  addr.Over(),
				UnderlayAddr: addr.Under(),
				HiveParams:   hp,
			}
			return network.NewBzz(config, kademlia(ctx.Config.ID), stateStore), nil
		},
	}
}
