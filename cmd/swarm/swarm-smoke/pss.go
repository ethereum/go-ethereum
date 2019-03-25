package main

import (
	"bytes"
	"context"
	"crypto/sha1"
	"errors"
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
}

type pssNode struct {
	hostIdx        int
	addr           []byte
	pubkey         string
	client         *rpc.Client
	deregisterFunc func()
	msgC           chan pss.APIMsg
}

type pssSession struct {
	topic        pss.Topic
	msgC         chan pss.APIMsg
	nodes        []*pssNode
	requiredJobs map[string]*pssJob
}

// pssAsymCheck sends one or more messages (depending on pssMessageCount)
// using asymmetric encryption across random nodes.
func pssAsymCheck(ctx *cli.Context, tuid string) error {

	// use input seed if it has been set
	if inputSeed != 0 {
		seed = inputSeed
	}
	rand.Seed(int64(seed))
	if pssMessageCount <= 0 {
		pssMessageCount = 1
		log.Warn(fmt.Sprintf("message count should be a positive number. Defaulting to %d", pssMessageCount))
	}
	log.Info("pss-asym test started", "msgCount", pssMessageCount)

	errc := make(chan error)
	session := pssSetup()
	if len(session.nodes) <= 1 {
		return errors.New("at least 2 nodes are required to be working")
	}
	defer func() {
		for _, n := range session.nodes {
			n.deregisterFunc()
		}
	}()

	go func() {
		var failCount, successCount int64
		for i := 0; i < pssMessageCount; i++ {
			err := pssAsymDo(ctx, session, tuid)
			if err != nil {
				failCount++
				log.Error("error sending pss msg", "err", err)
			} else {
				successCount++
			}
		}

		metrics.GetOrRegisterCounter("pss.asym.failMsg", nil).Inc(failCount)
		metrics.GetOrRegisterCounter("pss.asym.successMsg", nil).Inc(successCount)

		log.Info("pss-asym test ended", "success", successCount, "failures", failCount)

		if failCount > 0 {
			errc <- errors.New("some messages were not delivered")
		} else {
			errc <- nil
		}
	}()

	select {
	case err := <-errc:
		if err != nil {
			metrics.GetOrRegisterCounter("pss.asym.fail", nil).Inc(1)
		}
		return err
	case <-time.After(time.Duration(timeout) * time.Second):
		metrics.GetOrRegisterCounter("pss.asym.timeout", nil).Inc(1)
		return fmt.Errorf("timeout after %v sec", timeout)
	}

}

func pssSetup() *pssSession {
	// random topic, one per session,  same for all msgs
	topic := pss.BytesToTopic(testutil.RandomBytes(seed, 4))

	log.Trace("pss random topic", "topic", topic.String())

	session := &pssSession{
		msgC:         make(chan pss.APIMsg),
		requiredJobs: make(map[string]*pssJob),
		topic:        topic,
	}

	// set up the necessary info for each pss node
	for i, host := range hosts {
		httpHost := fmt.Sprintf("ws://%s:%d", host, 8546)
		rpcClient, err := rpc.Dial(httpHost)
		if err != nil {
			log.Error("Error dialing host", "err", err)
			continue
		}

		// get overlay address
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
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		sub, err := rpcClient.Subscribe(ctx, "pss", session.msgC, "receive", session.topic, true, false)
		if err != nil {
			log.Error("Error calling host for subscribe", "err", err)
			continue
		}
		session.nodes = append(session.nodes,
			&pssNode{
				hostIdx:        i,
				addr:           []byte(addr),
				client:         rpcClient,
				pubkey:         pubkey,
				deregisterFunc: sub.Unsubscribe,
			},
		)
	}

	return session
}

// pssAsymDo sends a single PSS message between two random nodes using asymetric encryption
func pssAsymDo(ctx *cli.Context, session *pssSession, tuid string) error {

	senderNodeIdx := rand.Intn(len(session.nodes))
	senderNode := session.nodes[senderNodeIdx]
	log.Trace("sender node", "pss_baseAddr", hexutil.Encode(senderNode.addr), "host", hosts[senderNodeIdx])

	// receiving node has to be different than sender
	recvNodeIdx := senderNodeIdx
	for recvNodeIdx == senderNodeIdx {
		recvNodeIdx = rand.Intn(len(session.nodes))
	}
	recvNode := session.nodes[recvNodeIdx]
	log.Trace("recv node", "pss_baseAddr", hexutil.Encode(recvNode.addr), "host", hosts[recvNodeIdx])

	// set recipient in pivot node
	err := senderNode.client.Call(nil, "pss_setPeerPublicKey", recvNode.pubkey, session.topic, "0x")
	if err != nil {
		return err
	}

	// create new message and add it to job index to check for receives
	randomMsg := testutil.RandomBytes(seed, 128)
	// change seed so that the next random message is different
	seed = seed + 1

	msgIdx := toMsgIdx(randomMsg)
	session.requiredJobs[msgIdx] = &pssJob{
		msg:  randomMsg,
		mode: pssModeAsym,
	}

	// send the message
	hostIdx := session.nodes[recvNodeIdx].hostIdx
	log.Debug("sending msg", "job", hexutil.Encode([]byte(msgIdx)), "sender", hosts[senderNodeIdx], "recv", hosts[hostIdx])
	err = senderNode.client.Call(nil, "pss_sendAsym", recvNode.pubkey, session.topic, hexutil.Encode(randomMsg))
	if err != nil {
		return err
	}

	// receive the message and check for its content
	select {
	case res := <-session.msgC:
		resIdx := toMsgIdx(res.Msg)
		resJob, ok := session.requiredJobs[resIdx]
		if !ok {
			return fmt.Errorf("corrupt message for job %s", resIdx)
		}
		if !bytes.Equal(res.Msg, resJob.msg) {
			return fmt.Errorf("message mismatch. expected: %s got: %s", resJob.msg, res.Msg)
		}
		log.Debug("got msg", "job", hexutil.Encode([]byte(resIdx)))
	case <-time.After(1 * time.Second):
		return errors.New("msg timeout after 1 sec")
	}
	return nil
}

func toMsgIdx(msg []byte) string {
	h := sha1.New()
	h.Write(msg)
	return string(h.Sum(nil))
}
