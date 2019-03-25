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

type pssJob struct {
	sender   *pssNode
	receiver *pssNode
	msg      []byte
}

type pssNode struct {
	hostIdx        int
	addr           []byte
	pubkey         string
	client         *rpc.Client
	deregisterFunc func()
}

type pssSession struct {
	topic pss.Topic
	msgC  chan pss.APIMsg
	nodes []*pssNode
	jobs  map[string]*pssJob
}

type pssTestFn func(ctx *cli.Context, session *pssSession, tuid string) error

func pssAsymCheck(ctx *cli.Context, tuid string) error {
	return pssCheck(ctx, tuid, "asym", pssAsymDo)
}

func pssSymCheck(ctx *cli.Context, tuid string) error {
	return pssCheck(ctx, tuid, "sym", pssSymDo)
}
func pssRawCheck(ctx *cli.Context, tuid string) error {
	return pssCheck(ctx, tuid, "raw", pssRawDo)
}

func pssCheck(ctx *cli.Context, tuid string, tag string, fn pssTestFn) error {
	// use input seed if it has been set
	if inputSeed != 0 {
		seed = inputSeed
	}
	rand.Seed(int64(seed))
	if pssMessageCount <= 0 {
		pssMessageCount = 1
		log.Warn(fmt.Sprintf("message count should be a positive number. Defaulting to %d", pssMessageCount))
	}
	log.Info(fmt.Sprintf("pss.%s test started", tag), "msgCount", pssMessageCount)

	session := pssSetup()
	if len(session.nodes) <= 1 {
		return errors.New("at least 2 nodes are required to be working")
	}
	defer func() {
		for _, n := range session.nodes {
			n.deregisterFunc()
		}
	}()

	errc := make(chan error)
	go func() {
		var failCount, successCount int64
		for i := 0; i < pssMessageCount; i++ {
			err := fn(ctx, session, tuid)
			if err != nil {
				failCount++
				log.Error("error sending pss msg", "err", err)
			} else {
				successCount++
			}
		}

		metrics.GetOrRegisterCounter(fmt.Sprintf("pss.%s.failMsg", tag), nil).Inc(failCount)
		metrics.GetOrRegisterCounter(fmt.Sprintf("pss.%s.successMsg", tag), nil).Inc(successCount)

		log.Info(fmt.Sprintf("pss.%s test ended", tag), "success", successCount, "failures", failCount)

		if failCount > 0 {
			errc <- errors.New("some messages were not delivered")
		} else {
			errc <- nil
		}
	}()

	select {
	case err := <-errc:
		if err != nil {
			metrics.GetOrRegisterCounter(fmt.Sprintf("pss.%s.fail", tag), nil).Inc(1)
		}
		return err
	case <-time.After(time.Duration(timeout) * time.Second):
		metrics.GetOrRegisterCounter(fmt.Sprintf("pss.%s.timeout", tag), nil).Inc(1)
		return fmt.Errorf("timeout after %v sec", timeout)
	}

}

func pssSetup() *pssSession {
	// random topic, one per session,  same for all msgs
	topic := pss.BytesToTopic(testutil.RandomBytes(seed, 4))

	log.Trace("pss random topic", "topic", topic.String())

	session := &pssSession{
		msgC:  make(chan pss.APIMsg),
		jobs:  make(map[string]*pssJob),
		topic: topic,
	}

	// set up the necessary info for each pss node
	for i, host := range hosts {
		wsHost := wsEndpoint(host)

		rpcClient, err := rpc.Dial(wsHost)
		if err != nil {
			log.Error("error dialing host", "err", err)
			continue
		}

		// get overlay address
		var addr hexutil.Bytes
		err = rpcClient.Call(&addr, "pss_baseAddr")
		if err != nil {
			log.Error("error calling host for addr", "err", err)
			continue
		}

		// get pss public key
		var pubkey string
		err = rpcClient.Call(&pubkey, "pss_getPublicKey")
		if err != nil {
			log.Error("error calling host for pubkey", "err", err)
			continue
		}

		// subscribe to the topic for the session
		// this creates the incoming message handler automatically
		// any message received with this topic comes on the channel
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		sub, err := rpcClient.Subscribe(ctx, "pss", session.msgC, "receive", session.topic, true, false)
		if err != nil {
			log.Error("error calling host for subscribe", "err", err)
			continue
		}

		var dFn = func() {
			sub.Unsubscribe()
			rpcClient.Close()
		}
		session.nodes = append(session.nodes,
			&pssNode{
				hostIdx:        i,
				addr:           []byte(addr),
				client:         rpcClient,
				pubkey:         pubkey,
				deregisterFunc: dFn,
			},
		)
	}
	return session
}

// genJob generates a random message that will be sent
// from a random sending node to a random receiving node
func (s *pssSession) genJob() pssJob {
	senderNodeIdx := rand.Intn(len(s.nodes))
	senderNode := s.nodes[senderNodeIdx]
	log.Trace("sender node", "pss_baseAddr", hexutil.Encode(senderNode.addr), "host", hosts[senderNodeIdx])

	// receiving node has to be different than sender
	recvNodeIdx := senderNodeIdx
	for recvNodeIdx == senderNodeIdx {
		recvNodeIdx = rand.Intn(len(s.nodes))
	}
	recvNode := s.nodes[recvNodeIdx]
	log.Trace("recv node", "pss_baseAddr", hexutil.Encode(recvNode.addr), "host", hosts[recvNodeIdx])

	// create new message and add it to job index to check for receives
	randomMsg := testutil.RandomBytes(seed, 128)
	// change seed so that the next random message is different
	seed = seed + 1

	j := pssJob{
		sender:   senderNode,
		receiver: recvNode,
		msg:      randomMsg,
	}

	msgIdx := toMsgIdx(randomMsg)
	s.jobs[msgIdx] = &j

	log.Debug("generated job", "job", hexutil.Encode([]byte(msgIdx)), "sender", hosts[j.sender.hostIdx], "recv", hosts[j.receiver.hostIdx])

	return j
}

// waitForJob blocks until a msg is received or a timeout is reached
func (s *pssSession) waitForMsg() error {
	select {
	case res := <-s.msgC:
		resIdx := toMsgIdx(res.Msg)
		resJob, ok := s.jobs[resIdx]
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

// pssSymDo sends a single PSS message between two random nodes using symmetric encryption
func pssSymDo(ctx *cli.Context, session *pssSession, tuid string) error {
	j := session.genJob()

	symkey := make([]byte, 32)
	c, err := rand.Read(symkey)
	if err != nil {
		return err
	} else if c < 32 {
		return fmt.Errorf("symkey size mismatch, expected 32 got %d", c)
	}

	var senderSymKeyID string
	err = j.sender.client.Call(&senderSymKeyID, "pss_setSymmetricKey", symkey, session.topic, hexutil.Encode(j.receiver.addr), true)
	if err != nil {
		log.Error("error setting sym key on the sender", "err", err)
		return err
	}

	var recvSymKeyID string
	err = j.receiver.client.Call(&recvSymKeyID, "pss_setSymmetricKey", symkey, session.topic, hexutil.Encode(j.sender.addr), true)
	if err != nil {
		log.Error("error setting sym key on the receiver", "err", err)
		return err
	}

	err = j.sender.client.Call(nil, "pss_sendSym", senderSymKeyID, session.topic, hexutil.Encode(j.msg))
	if err != nil {
		log.Error("error sending message using sym encryption", "err", err)
		return err
	}

	return session.waitForMsg()
}

// pssAsymDo sends a single PSS message between two random nodes using asymmetric encryption
func pssAsymDo(ctx *cli.Context, session *pssSession, tuid string) error {
	j := session.genJob()

	err := j.sender.client.Call(nil, "pss_sendAsym", j.receiver.pubkey, session.topic, hexutil.Encode(j.msg))
	if err != nil {
		log.Error("error sending message using asym encryption", "err", err)
		return err
	}

	return session.waitForMsg()
}

func pssRawDo(ctx *cli.Context, session *pssSession, tuid string) error {
	j := session.genJob()

	err := j.sender.client.Call(nil, "pss_sendRaw", hexutil.Encode(j.receiver.addr), session.topic, hexutil.Encode(j.msg))
	if err != nil {
		log.Error("error sending raw message", "err", err)
		return err
	}

	return session.waitForMsg()
}

func toMsgIdx(msg []byte) string {
	h := sha1.New()
	h.Write(msg)
	return string(h.Sum(nil))
}
