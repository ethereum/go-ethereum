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
	ping := &pss.Ping{
		C: make(chan struct{}),
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
	ping := &pss.Ping{
		C: make(chan struct{}),
	}
	proto := newProtocol(ping)
	client, err := baseTester(t, proto, ps, ctx, cancel, quitC)
	if err != nil {
		t.Fatalf(err.Error())
	}

	client.rpc.Call(&addr, "psstest_baseAddr")

	code, _ := pss.PingProtocol.GetCode(&pss.PingMsg{})
	rlpbundle, err := pss.NewProtocolMsg(code, &pss.PingMsg{
		Created: time.Now(),
	})
	if err != nil {
		t.Fatalf("couldn't make pssmsg: %v", err)
	}

	pssenv := pss.NewEnvelope(addr, pss.NewTopic(proto.Name, int(proto.Version)), rlpbundle)
	pssmsg := pss.PssMsg{
		To:      addr,
		Payload: pssenv,
	}

	ps.Process(&pssmsg)

	select {
	case <-client.ctx.Done():
		t.Fatalf("outgoing timed out or canceled")
	case <-ping.C:
	}

	quitC <- struct{}{}
}

func TestOutgoing(t *testing.T) {
	quitC := make(chan struct{})
	ps := pss.NewTestPss(nil)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*250)
	var addr []byte
	var potaddr pot.Address

	ping := &pss.Ping{
		C: make(chan struct{}),
	}
	proto := newProtocol(ping)
	client, err := baseTester(t, proto, ps, ctx, cancel, quitC)
	if err != nil {
		t.Fatalf(err.Error())
	}

	client.rpc.Call(&addr, "psstest_baseAddr")
	copy(potaddr[:], addr)

	msg := &pss.PingMsg{
		Created: time.Now(),
	}

	topic := pss.NewTopic(pss.PingProtocol.Name, int(pss.PingProtocol.Version))
	client.AddPssPeer(potaddr, pss.PingProtocol)
	nid, _ := discover.HexID("0x00")
	p := p2p.NewPeer(nid, fmt.Sprintf("%v", potaddr), []p2p.Cap{})
	pp := protocols.NewPeer(p, client.peerPool[topic][potaddr], pss.PingProtocol)
	pp.Send(msg)
	select {
	case <-client.ctx.Done():
		t.Fatalf("outgoing timed out or canceled")
	case <-ping.C:
	}
	quitC <- struct{}{}
}

func baseTester(t *testing.T, proto *p2p.Protocol, ps *pss.Pss, ctx context.Context, cancel func(), quitC chan struct{}) (*Client, error) {
	var err error

	client := newTestclient(t, ctx, cancel, quitC)

	err = client.RunProtocol(proto)

	if err != nil {
		return nil, err
	}

	return client, nil
}

func newProtocol(ping *pss.Ping) *p2p.Protocol {

	return &p2p.Protocol{
		Name:    pss.PingProtocol.Name,
		Version: pss.PingProtocol.Version,
		Length:  1,
		Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
			pp := protocols.NewPeer(p, rw, pss.PingProtocol)
			pp.Run(ping.PingHandler)
			return nil
		},
	}
}

func newTestclient(t *testing.T, ctx context.Context, cancel func(), quitC chan struct{}) *Client {

	ps := pss.NewTestPss(nil)
	srv := rpc.NewServer()
	srv.RegisterName("pss", pss.NewAPI(ps))
	srv.RegisterName("psstest", pss.NewAPITest(ps))
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

	pssclient, err := NewClient(ctx, cancel, "ws://localhost:8546")
	if err != nil {
		t.Fatalf(err.Error())
	}

	return pssclient
}
