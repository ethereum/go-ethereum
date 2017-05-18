package pss

import (
	"context"
	"fmt"
	"os"
	"net"
	"net/http"
	"testing"
	"time"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/pot"
	"github.com/ethereum/go-ethereum/rpc"
)

func init() {
	h := log.CallerFileHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(true)))
	log.Root().SetHandler(h)
}

func TestRunProtocol(t *testing.T) {
	quitC := make(chan struct{})
	pss := newTestPss(nil)
	ping := &pssPing{
		quitC: make(chan struct{}),
	}
	proto := newProtocol(ping)	
	_, err := baseTester(t, proto, pss, nil, nil, quitC)
	if err != nil {
		t.Fatalf(err.Error())
	}
	quitC <- struct{}{}
}

func TestIncoming(t *testing.T) {
	quitC := make(chan struct{})
	pss := newTestPss(nil)
	ctx, cancel := context.WithCancel(context.Background())
	var addr []byte
	ping := &pssPing{
		quitC: make(chan struct{}),
	}
	proto := newProtocol(ping)	
	client, err := baseTester(t, proto, pss, ctx, cancel, quitC)
	if err != nil {
		t.Fatalf(err.Error())
	}
	
	client.ws.Call(&addr, "pss_baseAddr")

	code, _ := pssPingProtocol.GetCode(&pssPingMsg{})
	rlpbundle, err := newProtocolMsg(code, &pssPingMsg{
		Created: time.Now(),
	})
	if err != nil {
		t.Fatalf("couldn't make pssmsg")
	}

	pssenv := PssEnvelope{
		From: addr,
		Topic:       NewTopic(proto.Name, int(proto.Version)),
		TTL:         DefaultTTL,
		Payload:     rlpbundle,
	}
	pssmsg := PssMsg{
		To: addr,
		Payload: &pssenv,
	}
	
	pss.Process(&pssmsg)
	
	go func() {
		<-ping.quitC
		client.cancel()
	}()
	
	<-client.ctx.Done()
	quitC <- struct{}{}
}

func TestOutgoing(t *testing.T) {
	quitC := make(chan struct{})
	pss := newTestPss(nil)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond * 250)
	var addr []byte
	var potaddr pot.Address
	
	ping := &pssPing{
		quitC: make(chan struct{}),
	}
	proto := newProtocol(ping)	
	client, err := baseTester(t, proto, pss, ctx, cancel, quitC)
	if err != nil {
		t.Fatalf(err.Error())
	}
	
	client.ws.Call(&addr, "pss_baseAddr")
	copy(potaddr[:], addr)
					
	msg := &pssPingMsg{
		Created: time.Now(),
	}
	
	topic := NewTopic(pssPingProtocol.Name, int(pssPingProtocol.Version))
	client.AddPssPeer(potaddr, pssPingProtocol)
	nid, _ := discover.HexID("0x00")
	p := p2p.NewPeer(nid, fmt.Sprintf("%v", potaddr), []p2p.Cap{})
	pp := protocols.NewPeer(p, client.peerPool[topic][potaddr], pssPingProtocol)
	pp.Send(msg)
	<-client.ctx.Done()
	quitC <- struct{}{}
}

func baseTester(t *testing.T, proto *p2p.Protocol, pss *Pss, ctx context.Context, cancel func(), quitC chan struct{}) (*PssClient, error) {
	var err error
	
	client := newClient(t, pss, ctx, cancel, quitC)
	
	err = client.Start()
	if err != nil {
		return nil, err
	}
	
	err = client.RunProtocol(proto, pssPingProtocol)
	
	if err != nil {
		return nil, err
	}
	
	return client, nil
}

func newProtocol(ping *pssPing) *p2p.Protocol {
	
	return &p2p.Protocol{
		Name: pssPingProtocol.Name,
		Version: pssPingProtocol.Version,
		Length: 1,
		Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
			pp := protocols.NewPeer(p, rw, pssPingProtocol)
			pp.Run(ping.pssPingHandler)
			return nil
		},
	}
}

func newClient(t *testing.T, pss *Pss, ctx context.Context, cancel func(), quitC chan struct{}) *PssClient {
	pssclient := NewPssClient(ctx, cancel, "", 0, false, "")
	
	srv := rpc.NewServer()
	srv.RegisterName("pss", NewPssAPI(pss))
	ws := srv.WebsocketHandler([]string{"*"})
	uri := fmt.Sprintf("%s:%d", node.DefaultWSHost, node.DefaultWSPort)
	
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
