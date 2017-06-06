package client

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/pot"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/pss"
)

func init() {
	h := log.CallerFileHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(true)))
	log.Root().SetHandler(h)
}

func TestRunProtocol(t *testing.T) {
	quitC := make(chan struct{})
	ps := pss.NewTestPss(nil)
	ping := &pss.PssPing{
		QuitC: make(chan struct{}),
	}
	proto := newProtocol(ping)
	_, err := baseTester(t, proto, ps, nil, nil, quitC)
	if err != nil {
		t.Fatalf(err.Error())
	}
	quitC <- struct{}{}
}

func TestIncoming(t *testing.T) {
	quitC := make(chan struct{})
	ps := pss.NewTestPss(nil)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	var addr []byte
	ping := &pss.PssPing{
		QuitC: make(chan struct{}),
	}
	proto := newProtocol(ping)
	client, err := baseTester(t, proto, ps, ctx, cancel, quitC)
	if err != nil {
		t.Fatalf(err.Error())
	}

	client.ws.Call(&addr, "psstest_baseAddr")

	code, _ := pss.PssPingProtocol.GetCode(&pss.PssPingMsg{})
	rlpbundle, err := pss.NewProtocolMsg(code, &pss.PssPingMsg{
		Created: time.Now(),
	})
	if err != nil {
		t.Fatalf("couldn't make pssmsg")
	}

	pssenv := pss.PssEnvelope{
		From:    addr,
		Topic:   pss.NewTopic(proto.Name, int(proto.Version)),
		TTL:     pss.DefaultTTL,
		Payload: rlpbundle,
	}
	pssmsg := pss.PssMsg{
		To:      addr,
		Payload: &pssenv,
	}

	ps.Process(&pssmsg)

	go func() {
		<-ping.QuitC
		client.cancel()
	}()

	select {
	case <-client.ctx.Done():
		t.Fatalf("outgoing timed out or canceled")
	default:
	}
	quitC <- struct{}{}
}

func TestOutgoing(t *testing.T) {
	quitC := make(chan struct{})
	ps := pss.NewTestPss(nil)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*250)
	var addr []byte
	var potaddr pot.Address

	ping := &pss.PssPing{
		QuitC: make(chan struct{}),
	}
	proto := newProtocol(ping)
	client, err := baseTester(t, proto, ps, ctx, cancel, quitC)
	if err != nil {
		t.Fatalf(err.Error())
	}

	client.ws.Call(&addr, "psstest_baseAddr")
	copy(potaddr[:], addr)

	msg := &pss.PssPingMsg{
		Created: time.Now(),
	}

	topic := pss.NewTopic(pss.PssPingProtocol.Name, int(pss.PssPingProtocol.Version))
	client.AddPssPeer(potaddr, pss.PssPingProtocol)
	nid, _ := discover.HexID("0x00")
	p := p2p.NewPeer(nid, fmt.Sprintf("%v", potaddr), []p2p.Cap{})
	pp := protocols.NewPeer(p, client.peerPool[topic][potaddr], pss.PssPingProtocol)
	pp.Send(msg)
	select {
	case <-client.ctx.Done():
		t.Fatalf("outgoing timed out or canceled")
	default:
	}
	quitC <- struct{}{}
}

func baseTester(t *testing.T, proto *p2p.Protocol, ps *pss.Pss, ctx context.Context, cancel func(), quitC chan struct{}) (*PssClient, error) {
	var err error

	client := newClient(t, ctx, cancel, quitC)

	err = client.Start()
	if err != nil {
		return nil, err
	}

	err = client.RunProtocol(proto)

	if err != nil {
		return nil, err
	}

	return client, nil
}

func newProtocol(ping *pss.PssPing) *p2p.Protocol {

	return &p2p.Protocol{
		Name:    pss.PssPingProtocol.Name,
		Version: pss.PssPingProtocol.Version,
		Length:  1,
		Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
			pp := protocols.NewPeer(p, rw, pss.PssPingProtocol)
			pp.Run(ping.PssPingHandler)
			return nil
		},
	}
}

func newClient(t *testing.T, ctx context.Context, cancel func(), quitC chan struct{}) *PssClient {

	conf := NewPssClientConfig()

	pssclient := NewPssClient(ctx, cancel, conf)

	ps := pss.NewTestPss(nil)
	srv := rpc.NewServer()
	srv.RegisterName("pss", pss.NewPssAPI(ps))
	srv.RegisterName("psstest", pss.NewPssAPITest(ps))
	ws := srv.WebsocketHandler([]string{"*"})
	uri := fmt.Sprintf("%s:%d", "localhost", 8546)

	sock, err := net.Listen("tcp", uri)
	if err != nil {
		t.Fatalf("Tcp (recv) on %s failed: %v", uri, err)
	}

	go func() {
		http.Serve(sock, ws)
	}()

	go func() {
		<-quitC
		sock.Close()
	}()
	return pssclient
}
