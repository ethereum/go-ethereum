package whisper

import (
	"bytes"
	"testing"
)

var topicCreationTests = []struct {
	data []byte
	hash [4]byte
}{
	{hash: [4]byte{0xc5, 0xd2, 0x46, 0x01}, data: nil},
	{hash: [4]byte{0xc5, 0xd2, 0x46, 0x01}, data: []byte{}},
	{hash: [4]byte{0x8f, 0x9a, 0x2b, 0x7d}, data: []byte("test name")},
}

func TestTopicCreation(t *testing.T) {
	for i, tt := range topicCreationTests {
		topic := NewTopic(tt.data)
		if bytes.Compare(topic[:], tt.hash[:]) != 0 {
			t.Errorf("test %d: hash mismatch: have %v, want %v.", i, topic, tt.hash)
		}
	}
}

func TestTopicSetCreation(t *testing.T) {
	topics := make([]Topic, len(topicCreationTests))
	for i, tt := range topicCreationTests {
		topics[i] = NewTopic(tt.data)
	}
	set := NewTopicSet(topics)
	for i, tt := range topicCreationTests {
		topic := NewTopic(tt.data)
		if _, ok := set[topic.String()]; !ok {
			t.Errorf("topic %d: not found in set", i)
		}
	}
}
