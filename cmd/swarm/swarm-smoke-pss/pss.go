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

type pssMode int

const (
	pssModeRaw = iota
	pssModeAsym
	pssModeSym
)

func (m pssMode) String() string {
	return [...]string{"raw", "asym", "sym"}[m]
}

type pssJob struct {
	sender   *pssNode
	receiver *pssNode
	mode     pssMode
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

func pssAsymCheck(ctx *cli.Context, tuid string) error {
	return runCheck(pssModeAsym, pssMessageCount)
}

func pssSymCheck(ctx *cli.Context, tuid string) error {
	return runCheck(pssModeSym, pssMessageCount)
}
func pssRawCheck(ctx *cli.Context, tuid string) error {
	return runCheck(pssModeRaw, pssMessageCount)
}

func runCheck(mode pssMode, count int) error {
	log.Info(fmt.Sprintf("pss.%s test started", mode), "msgCount", count)

	session := pssSetup()

	defer func() {
		for _, n := range session.nodes {
			n.deregisterFunc()
		}
	}()

	if len(session.nodes) <= 1 {
		return errors.New("at least 2 nodes are required to be working")
	}

	jobs := session.genJobs(count, mode)

	errc := make(chan error)
	go func() {
		var failCount, successCount int64
		sc, err := session.processJobs(jobs)
		if err != nil {
			log.Error("error processing some jobs", "err", err)
		}
		successCount = int64(sc)
		failCount = int64(count - sc)

		metrics.GetOrRegisterCounter(fmt.Sprintf("pss.%s.failMsg", mode), nil).Inc(failCount)
		metrics.GetOrRegisterCounter(fmt.Sprintf("pss.%s.successMsg", mode), nil).Inc(successCount)

		log.Info(fmt.Sprintf("pss.%s test ended", mode), "success", successCount, "failures", failCount)

		if failCount > 0 {
			errc <- errors.New("some messages were not delivered")
		} else {
			errc <- nil
		}
	}()

	select {
	case err := <-errc:
		if err != nil {
			metrics.GetOrRegisterCounter(fmt.Sprintf("pss.%s.fail", mode), nil).Inc(1)
		}
		return err
	case <-time.After(time.Duration(timeout) * time.Second):
		metrics.GetOrRegisterCounter(fmt.Sprintf("pss.%s.timeout", mode), nil).Inc(1)
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

// processJobs processes the given jobs and returns the success count and an error
func (s *pssSession) processJobs(jobs []pssJob) (int, error) {
	errs := []error{}
	for _, j := range jobs {
		var err error
		switch mode := j.mode; mode {
		case pssModeRaw:
			err = s.sendRawMessage(j.sender, j.receiver, j.msg)
			if err != nil {
				log.Error("error sending raw message", "err", err)
			}
		case pssModeSym:
			err = s.sendSymMessage(j.sender, j.receiver, j.msg)
			if err != nil {
				log.Error("error sending sym message", "err", err)
			}
		case pssModeAsym:
			err = s.sendAsymMessage(j.sender, j.receiver, j.msg)
			if err != nil {
				log.Error("error sending asym message", "err", err)
			}
		default:
			err = fmt.Errorf("invalid pssMode %d", mode)
			log.Error("error processing job type", "err", err)
		}

		err = s.waitForMsg()
		if err != nil {
			log.Error("error while waiting for msg", "err", err)
		}
		if err != nil {
			errs = append(errs, err)
		}

	}
	if len(errs) > 0 {
		return len(jobs) - len(errs), fmt.Errorf("%d/%d jobs failed processing", len(errs), len(jobs))
	}
	return len(jobs), nil
}

func (s *pssSession) genJobs(count int, mode pssMode) []pssJob {
	jobs := make([]pssJob, count)
	for i := 0; i < count; i++ {
		jobs[i] = s.genJob(mode)
	}
	return jobs
}

// genJob generates a pssJob with random message that will be sent
// from a random sending node to a random receiving node
func (s *pssSession) genJob(mode pssMode) pssJob {
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
	randomMsg := testutil.RandomBytes(seed, pssMessageSize)
	// change seed so that the next random message is different
	seed = seed + 1

	j := pssJob{
		sender:   senderNode,
		receiver: recvNode,
		mode:     mode,
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

func (s *pssSession) sendRawMessage(sender *pssNode, receiver *pssNode, msg []byte) error {
	return sender.client.Call(nil, "pss_sendRaw", hexutil.Encode(receiver.addr), s.topic, hexutil.Encode(msg))
}

func (s *pssSession) sendAsymMessage(sender *pssNode, receiver *pssNode, msg []byte) error {

	// share public keys between nodes
	err := sender.client.Call(nil, "pss_setPeerPublicKey", receiver.pubkey, s.topic, hexutil.Encode(receiver.addr))
	if err != nil {
		log.Error("error setting receivers public key on the sender side", "err", err)
		return err
	}
	err = receiver.client.Call(nil, "pss_setPeerPublicKey", sender.pubkey, s.topic, hexutil.Encode(sender.addr))
	if err != nil {
		log.Error("error setting senders public key on the receiver side", "err", err)
		return err
	}
	// send asym message
	return sender.client.Call(nil, "pss_sendAsym", receiver.pubkey, s.topic, hexutil.Encode(msg))
}

func (s *pssSession) sendSymMessage(sender *pssNode, receiver *pssNode, msg []byte) error {
	// create a shared secret for the symmetric encryption
	symkey := make([]byte, 32)
	c, err := rand.Read(symkey)
	if err != nil {
		return err
	} else if c < 32 {
		return fmt.Errorf("symkey size mismatch, expected 32 got %d", c)
	}

	// set the secret on both nodes
	var senderSymKeyID string
	err = sender.client.Call(&senderSymKeyID, "pss_setSymmetricKey", symkey, s.topic, hexutil.Encode(receiver.addr), true)
	if err != nil {
		log.Error("error setting sym key on the sender", "err", err)
		return err
	}

	var recvSymKeyID string
	err = receiver.client.Call(&recvSymKeyID, "pss_setSymmetricKey", symkey, s.topic, hexutil.Encode(sender.addr), true)
	if err != nil {
		log.Error("error setting sym key on the receiver", "err", err)
		return err
	}

	// send sym message
	return sender.client.Call(nil, "pss_sendSym", senderSymKeyID, s.topic, hexutil.Encode(msg))
}

func toMsgIdx(msg []byte) string {
	return fmt.Sprintf("%x", sha1.Sum(msg))
}
