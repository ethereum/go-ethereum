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
	// Create the topics individually
	for i, tt := range topicCreationTests {
		topic := NewTopic(tt.data)
		if bytes.Compare(topic[:], tt.hash[:]) != 0 {
			t.Errorf("binary test %d: hash mismatch: have %v, want %v.", i, topic, tt.hash)
		}
	}
	for i, tt := range topicCreationTests {
		topic := NewTopicFromString(string(tt.data))
		if bytes.Compare(topic[:], tt.hash[:]) != 0 {
			t.Errorf("textual test %d: hash mismatch: have %v, want %v.", i, topic, tt.hash)
		}
	}
	// Create the topics in batches
	binaryData := make([][]byte, len(topicCreationTests))
	for i, tt := range topicCreationTests {
		binaryData[i] = tt.data
	}
	textualData := make([]string, len(topicCreationTests))
	for i, tt := range topicCreationTests {
		textualData[i] = string(tt.data)
	}

	topics := NewTopics(binaryData...)
	for i, tt := range topicCreationTests {
		if bytes.Compare(topics[i][:], tt.hash[:]) != 0 {
			t.Errorf("binary batch test %d: hash mismatch: have %v, want %v.", i, topics[i], tt.hash)
		}
	}
	topics = NewTopicsFromStrings(textualData...)
	for i, tt := range topicCreationTests {
		if bytes.Compare(topics[i][:], tt.hash[:]) != 0 {
			t.Errorf("textual batch test %d: hash mismatch: have %v, want %v.", i, topics[i], tt.hash)
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
