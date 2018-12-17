package feed

import (
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/swarm/testutil"
)

func TestTopic(tx *testing.T) {
	t := testutil.BeginTest(tx, false) // set to true to generate results
	defer t.FinishTest()

	related, _ := hexutil.Decode("0xabcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789")
	topicName := "test-topic"
	topic, _ := NewTopic(topicName, related)
	hex := topic.Hex()
	t.EqualsKey("hex", hex)

	var topic2 Topic
	topic2.FromHex(hex)
	t.Equals(topic, topic2)
	t.Equals(topicName, topic2.Name(related))

	t.TestJSONMarshaller("topic.json", &topic2)
}
