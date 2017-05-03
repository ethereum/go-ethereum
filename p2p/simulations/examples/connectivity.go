package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/rpc"
)

// main() starts a simulation session which is capable of creating in-memory
// simulation networks containing nodes running a simple ping-pong protocol
func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	log.Root().SetHandler(log.LvlFilterHandler(log.LvlTrace, log.StreamHandler(os.Stderr, log.TerminalFormat(false))))

	services := map[string]adapters.ServiceFunc{
		"ping-pong": func(id *adapters.NodeId) node.Service {
			return newPingPongService(id)
		},
	}

	adapters.RegisterServices(services)

	config := &simulations.ServerConfig{}

	adapter := flag.String("adapter", "sim", `node adapter to use (one of "sim", "exec" or "docker")`)
	flag.Parse()

	switch *adapter {

	case "sim":
		log.Info("using sim adapter")
		config.Adapter = adapters.NewSimAdapter(services)

	case "exec":
		tmpdir, err := ioutil.TempDir("", "p2p-example")
		if err != nil {
			log.Crit("error creating temp dir", "err", err)
		}
		defer os.RemoveAll(tmpdir)
		log.Info("using exec adapter", "tmpdir", tmpdir)
		config.Adapter = adapters.NewExecAdapter(tmpdir)

	case "docker":
		log.Info("using docker adapter")
		var err error
		config.Adapter, err = adapters.NewDockerAdapter()
		if err != nil {
			log.Crit("error creating docker adapter", "err", err)
		}

	default:
		log.Crit(fmt.Sprintf("unknown node adapter %q", *adapter))
	}

	log.Info("starting simulation server on 0.0.0.0:8888...")
	http.ListenAndServe(":8888", simulations.NewServer(config))
}

// pingPongService runs a ping-pong protocol between nodes where each node
// sends a ping to all its connected peers every 10s and receives a pong in
// return
type pingPongService struct {
	id       *adapters.NodeId
	log      log.Logger
	received int64
}

func newPingPongService(id *adapters.NodeId) *pingPongService {
	return &pingPongService{
		id:  id,
		log: log.New("node.id", id),
	}
}

func (p *pingPongService) Protocols() []p2p.Protocol {
	return []p2p.Protocol{{
		Name:     "ping-pong",
		Version:  1,
		Length:   2,
		Run:      p.Run,
		NodeInfo: p.Info,
	}}
}

func (p *pingPongService) APIs() []rpc.API {
	return nil
}

func (p *pingPongService) Start(server p2p.Server) error {
	p.log.Info("ping-pong service starting")
	return nil
}

func (p *pingPongService) Stop() error {
	p.log.Info("ping-pong service stopping")
	return nil
}

func (p *pingPongService) Info() interface{} {
	return struct {
		Received int64 `json:"received"`
	}{
		atomic.LoadInt64(&p.received),
	}
}

const (
	pingMsgCode = iota
	pongMsgCode
)

// Run implements the ping-pong protocol which sends ping messages to the peer
// at 10s intervals, and responds to pings with pong messages.
func (p *pingPongService) Run(peer *p2p.Peer, rw p2p.MsgReadWriter) error {
	log := p.log.New("peer.id", peer.ID())

	errC := make(chan error)
	go func() {
		for range time.Tick(10 * time.Second) {
			log.Info("sending ping")
			if err := p2p.Send(rw, pingMsgCode, "PING"); err != nil {
				errC <- err
				return
			}
		}
	}()
	go func() {
		for {
			msg, err := rw.ReadMsg()
			if err != nil {
				errC <- err
				return
			}
			payload, err := ioutil.ReadAll(msg.Payload)
			if err != nil {
				errC <- err
				return
			}
			log.Info("received message", "msg.code", msg.Code, "msg.payload", string(payload))
			atomic.AddInt64(&p.received, 1)
			if msg.Code == pingMsgCode {
				log.Info("sending pong")
				go p2p.Send(rw, pongMsgCode, "PONG")
			}
		}
	}()
	return <-errC
}
