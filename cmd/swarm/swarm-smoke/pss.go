package main

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/pss"
	"github.com/ethereum/go-ethereum/swarm/testutil"

	cli "gopkg.in/urfave/cli.v1"
)

const (
	pssModeRaw = iota
	pssModeAsym
	pssModeSym
)

type pssJob struct {
	msg  []byte
	mode int
	prox bool
}

type pssNode struct {
	hostIdx        int
	addr           []byte
	pubkey         string
	client         *rpc.Client
	deregisterFunc func()
	msgC           chan pss.APIMsg
}

// keyed by hex representation of overlay address
type pssSession struct {
	topic        pss.Topic
	msgC         chan pss.APIMsg
	nodes        []*pssNode
	requiredJobs map[string]*pssJob
	allowedJobs  map[string]*pssJob
}

func pssChecks(ctx *cli.Context, tuid string) error {
	return pssAsym(ctx, tuid)
}

func pssAsym(ctx *cli.Context, tuid string) error {

	errc := make(chan error)

	session := pssSetup()
	defer func() {
		for _, n := range session.nodes {
			n.deregisterFunc()
		}
	}()

	go func() {
		errc <- pssAsymDo(ctx, session, tuid)
	}()

	select {
	case err := <-errc:
		if err != nil {
			metrics.GetOrRegisterCounter(fmt.Sprintf("%s.fail", commandName), nil).Inc(1)
		}
		return err
	case <-time.After(time.Duration(timeout) * time.Second):
		metrics.GetOrRegisterCounter(fmt.Sprintf("%s.timeout", commandName), nil).Inc(1)

		e := fmt.Errorf("timeout after %v sec", timeout)
		return e
	}

}

func pssSetup() *pssSession {

	// random topic, one per session,  same for all msgs
	topic := pss.BytesToTopic(testutil.RandomBytes(seed, 4))

	session := &pssSession{
		msgC:         make(chan pss.APIMsg),
		requiredJobs: make(map[string]*pssJob),
		allowedJobs:  make(map[string]*pssJob),
		topic:        topic,
	}

	// hosts is global :/
	// set up the necessary info for each pss node
	for i, host := range hosts {
		httpHost := fmt.Sprintf("ws://%s:%d", host, 8546)
		rpcClient, err := rpc.Dial(httpHost)
		if err != nil {
			log.Error("Error dialing host", "err", err)
			continue
		}

		// get overlay address
		// TODO this should be done automatically for all nodes anyway in all smokes perhaps?
		var addr hexutil.Bytes
		err = rpcClient.Call(&addr, "pss_baseAddr")
		if err != nil {
			log.Error("Error calling host for addr", "err", err)
			continue
		}

		// get pss public key
		var pubkey string
		err = rpcClient.Call(&pubkey, "pss_getPublicKey")
		if err != nil {
			log.Error("Error calling host for pubkey", "err", err)
			continue
		}

		// subscribe to the topic for the session
		// this creates the incoming message handler automatically
		// any message received with this topic comes on the channel
		//msgC := make(chan pss.APIMsg)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		sub, err := rpcClient.Subscribe(ctx, "pss", session.msgC, "receive", session.topic, true, false)
		if err != nil {
			log.Error("Error calling host for subscribe", "err", err)
			continue
		}
		session.nodes = append(session.nodes,
			&pssNode{
				hostIdx: i,
				//msgC:           msgC,
				addr:           []byte(addr),
				client:         rpcClient,
				pubkey:         pubkey,
				deregisterFunc: sub.Unsubscribe,
			},
		)
	}

	return session
}

func pssAsymDo(ctx *cli.Context, session *pssSession, tuid string) error {

	// can we choose? if so, change
	senderNode := session.nodes[0]

	// add if single blah blah...
	recvNodeIdx := rand.Intn(len(session.nodes)-1) + 1
	recvNode := session.nodes[recvNodeIdx]

	// set recipient in pivot node
	err := senderNode.client.Call(nil, "pss_setPeerPublicKey", recvNode.pubkey, session.topic, "0x")
	if err != nil {
		return err
	}

	// for asym it's not necessary to set recipient, but for sym we must
	//	err = recvNode.client.Call(nil, "pss_setPeerPublicKey", senderNode.pubkey, session.topic, "0x")
	//	if err != nil {
	//		return err
	//	}

	// create new message and add it to job index to check for receives
	randomMsg := testutil.RandomBytes(seed, 128)
	msgIdx := toMsgIdx(randomMsg)
	session.requiredJobs[msgIdx] = &pssJob{
		msg:  randomMsg,
		mode: pssModeAsym,
	}

	// send the msg
	hostIdx := session.nodes[recvNodeIdx].hostIdx
	log.Info("sending pss", "sender", hosts[0], "recv", hosts[hostIdx])
	err = senderNode.client.Call(nil, "pss_sendAsym", recvNode.pubkey, session.topic, hexutil.Encode(randomMsg))
	if err != nil {
		return err
	}

	// receive the msg or timeout and fail
	// and check its validity
	pctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()
	select {
	//case res := <-(recvNode.msgC):
	case res := <-(session.msgC):
		resIdx := toMsgIdx(res.Msg)
		resJob, ok := session.requiredJobs[resIdx]
		if !ok {
			return fmt.Errorf("corrupt message", "job", resIdx)
		}
		if !bytes.Equal(res.Msg, resJob.msg) {
			return fmt.Errorf("message mismatch", "job", resIdx)
		}
		log.Info("got msg", "job", hexutil.Encode([]byte(resIdx)))
	case <-pctx.Done():
		return pctx.Err()
	}
	return nil
}

func toMsgIdx(msg []byte) string {
	h := sha1.New()
	h.Write(msg)
	return string(h.Sum(nil))
}
