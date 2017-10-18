package shhclient

import (
	"bytes"
	"context"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/whisper/whisperv5"
	"testing"
	"time"
)

func getIPCPath() string {
	return node.DefaultIPCEndpoint("geth")
}
func getClient() (*Client, error) {
	return Dial(getIPCPath())
}

func TestBasic(t *testing.T) {
	var id string = "test"
	c, err := getClient()
	if err != nil { //if geth not start,just skip test.
		t.Log("skip Client test because of that geth  is not started")
		return
	}
	defer func() {
		c.c.Close()
	}()
	ctx := context.Background()
	if err != nil {
		t.Error(err)
		return
	}
	version, err := c.Version(ctx)
	if err != nil {
		t.Error(err)
		return
	}
	if version != whisperv5.ProtocolVersionStr {
		t.Fatalf("wrong version: %d.", version)
	}
	_, err = c.Info(ctx)
	if err != nil {
		t.Error(err)
		return
	}
	exist, err := c.HasKeyPair(ctx, id)
	if err != nil {
		t.Fatalf("failed initial HasIdentity: %s.", err)
	}
	if exist {
		t.Fatalf("failed initial HasIdentity: false positive.")
	}
	id = "arbitrary text"
	id2 := "another arbitrary string"

	exist, err = c.HasSymmetricKey(ctx, id)
	if err != nil {
		t.Fatalf("failed HasSymKey: %s.", err)
	}
	if exist {
		t.Fatalf("failed HasSymKey: false positive.")
	}

	id, err = c.NewSymmetricKey(ctx)
	if err != nil {
		t.Fatalf("failed GenerateSymKey: %s.", err)
	}

	exist, err = c.HasSymmetricKey(ctx, id)
	if err != nil {
		t.Fatalf("failed HasSymKey(): %s.", err)
	}
	if !exist {
		t.Fatalf("failed HasSymKey(): false negative.")
	}

	var password = []byte("some stuff here")
	id, err = c.GenerateSymmetricKeyFromPassword(ctx, password)
	if err != nil {
		t.Fatalf("failed AddSymKey: %s.", err)
	}

	id2, err = c.GenerateSymmetricKeyFromPassword(ctx, password)
	if err != nil {
		t.Fatalf("failed AddSymKey: %s.", err)
	}

	exist, err = c.HasSymmetricKey(ctx, id2)
	if err != nil {
		t.Fatalf("failed HasSymKey(id2): %s.", err)
	}
	if !exist {
		t.Fatalf("failed HasSymKey(id2): false negative.")
	}

	k1, err := c.GetSymmetricKey(ctx, id)
	if err != nil {
		t.Fatalf("failed GetSymKey(id): %s.", err)
	}
	k2, err := c.GetSymmetricKey(ctx, id2)
	if err != nil {
		t.Fatalf("failed GetSymKey(id2): %s.", err)
	}

	if !bytes.Equal(k1, k2) {
		t.Fatalf("installed keys are not equal")
	}

	err = c.DeleteSymmetricKey(ctx, id)
	if err != nil {
		t.Fatalf("failed DeleteSymKey(id): %s.", err)
	}

	exist, err = c.HasSymmetricKey(ctx, id)
	if err != nil {
		t.Fatalf("failed HasSymKey(id): %s.", err)
	}
	if exist {
		t.Fatalf("failed HasSymKey(id): false positive.")
	}
}

func TestSubscribe(t *testing.T) {
	var err error
	var ctx = context.Background()

	c, err := getClient()
	if err != nil { //if geth not start,just skip test.
		t.Log("skip Client test because of that geth  is not started")
		return
	}
	defer func() {
		c.c.Close()
	}()
	symKeyID, err := c.NewSymmetricKey(ctx)
	if err != nil {
		t.Fatalf("failed to GenerateSymKey: %s.", err)
	}

	var f whisperv5.Criteria
	f.SymKeyID = symKeyID
	f.Topics = make([]whisperv5.TopicType, 2)
	f.Topics[0] = whisperv5.TopicType{0xf8, 0xe9, 0xa0, 0xba}
	f.Topics[1] = whisperv5.TopicType{0xcb, 0x3c, 0xdd, 0xee}
	ch := make(chan *whisperv5.Message)
	sub, err := c.SubscribeMessages(ctx, f, ch)
	if err != nil {
		t.Fatalf("failed to subscribe: %s.", err)
	}
	sub.Unsubscribe()
	close(ch)
}

func TestIntegrationSymWithFilter(t *testing.T) {
	c, err := getClient()
	if err != nil {
		t.Log("skip Client test because of that geth  is not started")
		return
	}
	defer func() {
		c.c.Close()
	}()
	ctx := context.Background()
	symKeyID, err := c.NewSymmetricKey(ctx)
	if err != nil {
		t.Fatalf("failed to GenerateSymKey: %s.", err)
	}

	sigKeyID, err := c.NewKeyPair(ctx)
	if err != nil {
		t.Fatalf("failed NewIdentity: %s.", err)
	}
	if len(sigKeyID) == 0 {
		t.Fatalf("wrong signature.")
	}

	exist, err := c.HasKeyPair(ctx, sigKeyID)
	if err != nil {
		t.Fatalf("failed HasIdentity: %s.", err)
	}
	if !exist {
		t.Fatalf("failed HasIdentity: does not exist.")
	}

	sigPubKey, err := c.PublicKey(ctx, sigKeyID)
	if err != nil {
		t.Fatalf("failed GetPublicKey: %s.", err)
	}

	var topics [2]whisperv5.TopicType
	topics[0] = whisperv5.TopicType{0x00, 0x7f, 0x80, 0xff}
	topics[1] = whisperv5.TopicType{0xf2, 0x6e, 0x77, 0x79}
	var f whisperv5.Criteria
	f.SymKeyID = symKeyID
	f.Topics = make([]whisperv5.TopicType, 2)
	f.Topics[0] = topics[0]
	f.Topics[1] = topics[1]
	f.MinPow = whisperv5.DefaultMinimumPoW / 2
	f.Sig = sigPubKey
	f.AllowP2P = false
	ch := make(chan *whisperv5.Message)
	sub, err := c.SubscribeMessages(ctx, f, ch)
	if err != nil {
		t.Fatalf("failed to create new filter: %s.", err)
	}
	defer func() {
		sub.Unsubscribe()
		close(ch)
	}()
	var p whisperv5.NewMessage
	p.TTL = 1
	p.SymKeyID = symKeyID
	p.Sig = sigKeyID
	p.Padding = []byte("test string")
	p.Payload = []byte("extended test string")
	p.PowTarget = whisperv5.DefaultMinimumPoW
	p.PowTime = 2
	p.Topic = whisperv5.TopicType{0xf2, 0x6e, 0x77, 0x79}

	err = c.Post(ctx, p)
	if err != nil {
		t.Fatalf("failed to post message: %s.", err)
	}

	var msg *whisperv5.Message = nil
	select {
	case <-time.After(time.Second * 10):
	case msg = <-ch:
	}
	if msg == nil {
		t.Fatalf("failed to GetFilterChanges: got  messages.")
	}

	text := string(msg.Payload)
	if text != string("extended test string") {
		t.Fatalf("failed to decrypt first message: %s.", text)
	}
}
