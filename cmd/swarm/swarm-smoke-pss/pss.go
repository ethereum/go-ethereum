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

func pssAsymCheck(ctx *cli.Context) error {
	return runCheck(pssModeAsym, pssMessageCount, pssMessageSize)
}

func pssSymCheck(ctx *cli.Context) error {
	return runCheck(pssModeSym, pssMessageCount, pssMessageSize)
}

func pssRawCheck(ctx *cli.Context) error {
	return runCheck(pssModeRaw, pssMessageCount, pssMessageSize)
}

func pssAllCheck(ctx *cli.Context) error {
	gotErr := false
	if err := pssRawCheck(ctx); err != nil {
		log.Error("error when running raw tests", "err", err)
		gotErr = true
	}
	if err := pssSymCheck(ctx); err != nil {
		log.Error("error when running sym tests", "err", err)
		gotErr = true
	}
	if err := pssAsymCheck(ctx); err != nil {
		log.Error("error when running asym tests", "err", err)
		gotErr = true
	}
	if gotErr {
		return errors.New("some tests failed")
	}
	return nil
}

func runCheck(mode pssMode, count int, msgSizeBytes int) error {
	log.Info(fmt.Sprintf("pss.%s test started", mode), "msgCount", count, "msgBytes", msgSizeBytes)

	session := pssSetup()

	defer func() {
		for _, n := range session.nodes {
			n.deregisterFunc()
		}
	}()

	if len(session.nodes) <= 1 {
		return errors.New("at least 2 nodes are required to be working")
	}

	jobs := session.genJobs(count, mode, msgSizeBytes)

	errC := make(chan error)
	go func() {
		var failCount, successCount int64
		t := time.Now()
		sc, err := session.processJobs(jobs)
		if err != nil {
			log.Error("error processing some jobs", "err", err)
		}
		successCount = int64(sc)
		failCount = int64(count - sc)

		metrics.GetOrRegisterCounter(fmt.Sprintf("pss.%s.msgs.fail", mode), nil).Inc(failCount)
		metrics.GetOrRegisterCounter(fmt.Sprintf("pss.%s.msgs.success", mode), nil).Inc(successCount)

		totalTime := time.Since(t)

		metrics.GetOrRegisterResettingTimer(fmt.Sprintf("pss.%s.total-time", mode), nil).Update(totalTime)
		log.Info(fmt.Sprintf("pss.%s test ended", mode), "time", totalTime, "success", successCount, "failures", failCount)

		if failCount > 0 {
			errC <- errors.New("some messages were not delivered")
		} else {
			errC <- nil
		}
	}()

	select {
	case err := <-errC:
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
		wsHost := fmt.Sprintf("ws://%s:%d", host, wsPort)

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

func (s *pssSession) genJobs(count int, mode pssMode, msgSizeBytes int) []pssJob {
	jobs := make([]pssJob, count)
	for i := 0; i < count; i++ {
		jobs[i] = s.genJob(mode, msgSizeBytes)
	}
	return jobs
}

// genJob generates a pssJob with random message that will be sent
// from a random sending node to a random receiving node
func (s *pssSession) genJob(mode pssMode, msgSizeBytes int) pssJob {
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
	randomMsg := testutil.RandomBytes(seed, msgSizeBytes)
	// change seed so that the next random message is different
	seed++

	j := pssJob{
		sender:   senderNode,
		receiver: recvNode,
		mode:     mode,
		msg:      randomMsg,
	}

	msgIdx := toMsgIdx(randomMsg)
	s.jobs[msgIdx] = &j

	log.Debug("gen job", "job", hexutil.Encode([]byte(msgIdx)), "sender", hosts[j.sender.hostIdx], "recv", hosts[j.receiver.hostIdx])

	return j
}

// waitForMsg blocks until a msg is received or a timeout is reached
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
	case <-time.After(time.Duration(pssMessageTimeout) * time.Second):
		return fmt.Errorf("message timeout after %d sec", pssMessageTimeout)
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
