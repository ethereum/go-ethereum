// simple abstraction for implementing pss functionality
//
// the pss client library aims to simplify usage of the p2p.protocols package over pss
//
// IO is performed using the ordinary p2p.MsgReadWriter interface, which transparently communicates with a pss node via RPC using websockets as transport layer, using methods in the PssAPI class in the swarm/pss package
//
//
// Minimal-ish usage example (requires a running pss node with websocket RPC):
//
//
//   import (
//  	"context"
//  	"fmt"
//  	"os"
//  	pss "github.com/ethereum/go-ethereum/swarm/pss/client"
//  	"github.com/ethereum/go-ethereum/p2p/protocols"
//  	"github.com/ethereum/go-ethereum/p2p"
//  	"github.com/ethereum/go-ethereum/pot"
//  	"github.com/ethereum/go-ethereum/log"
//  )
//
//  type FooMsg struct {
//  	Bar int
//  }
//
//
//  func fooHandler (msg interface{}) error {
//  	foomsg, ok := msg.(*FooMsg)
//  	if ok {
//  		log.Debug("Yay, just got a message", "msg", foomsg)
//  	}
//  	return errors.New(fmt.Sprintf("Unknown message"))
//  }
//
//  spec := &protocols.Spec{
//  	Name: "foo",
//  	Version: 1,
//  	MaxMsgSize: 1024,
//  	Messages: []interface{}{
//  		FooMsg{},
//  	},
//  }
//
//  proto := &p2p.Protocol{
//  	Name: spec.Name,
//  	Version: spec.Version,
//  	Length: uint64(len(spec.Messages)),
//  	Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
//  		pp := protocols.NewPeer(p, rw, spec)
//  		return pp.Run(fooHandler)
//  	},
//  }
//
//  func implementation() {
//      cfg := pss.NewClientConfig()
//      psc := pss.NewClient(context.Background(), nil, cfg)
//      err := psc.Start()
//      if err != nil {
//      	log.Crit("can't start pss client")
//      	os.Exit(1)
//      }
//
//	log.Debug("connected to pss node", "bzz addr", psc.BaseAddr)
//
//      err = psc.RunProtocol(proto)
//      if err != nil {
//      	log.Crit("can't start protocol on pss websocket")
//      	os.Exit(1)
//      }
//
//      addr := pot.RandomAddress() // should be a real address, of course
//      psc.AddPssPeer(addr, spec)
//
//      // use the protocol for something
//
//      psc.Stop()
//  }
//
// BUG(test): TestIncoming test times out due to deadlock issues in the swarm hive
package client
