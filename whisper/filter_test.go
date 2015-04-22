package whisper

import (
	"bytes"

	"testing"
)

var filterTopicsCreationTests = []struct {
	topics [][]string
	filter [][][4]byte
}{
	{ // Simple topic filter
		topics: [][]string{
			{"abc", "def", "ghi"},
			{"def"},
			{"ghi", "abc"},
		},
		filter: [][][4]byte{
			{{0x4e, 0x03, 0x65, 0x7a}, {0x34, 0x60, 0x7c, 0x9b}, {0x21, 0x41, 0x7d, 0xf9}},
			{{0x34, 0x60, 0x7c, 0x9b}},
			{{0x21, 0x41, 0x7d, 0xf9}, {0x4e, 0x03, 0x65, 0x7a}},
		},
	},
	{ // Wild-carded topic filter
		topics: [][]string{
			{"abc", "def", "ghi"},
			{},
			{""},
			{"def"},
		},
		filter: [][][4]byte{
			{{0x4e, 0x03, 0x65, 0x7a}, {0x34, 0x60, 0x7c, 0x9b}, {0x21, 0x41, 0x7d, 0xf9}},
			{},
			{},
			{{0x34, 0x60, 0x7c, 0x9b}},
		},
	},
}

var filterTopicsCreationFlatTests = []struct {
	topics []string
	filter [][][4]byte
}{
	{ // Simple topic list
		topics: []string{"abc", "def", "ghi"},
		filter: [][][4]byte{
			{{0x4e, 0x03, 0x65, 0x7a}},
			{{0x34, 0x60, 0x7c, 0x9b}},
			{{0x21, 0x41, 0x7d, 0xf9}},
		},
	},
	{ // Wild-carded topic list
		topics: []string{"abc", "", "ghi"},
		filter: [][][4]byte{
			{{0x4e, 0x03, 0x65, 0x7a}},
			{},
			{{0x21, 0x41, 0x7d, 0xf9}},
		},
	},
}

func TestFilterTopicsCreation(t *testing.T) {
	// Check full filter creation
	for i, tt := range filterTopicsCreationTests {
		// Check the textual creation
		filter := NewFilterTopicsFromStrings(tt.topics...)
		if len(filter) != len(tt.topics) {
			t.Errorf("test %d: condition count mismatch: have %v, want %v", i, len(filter), len(tt.topics))
			continue
		}
		for j, condition := range filter {
			if len(condition) != len(tt.filter[j]) {
				t.Errorf("test %d, condition %d: size mismatch: have %v, want %v", i, j, len(condition), len(tt.filter[j]))
				continue
			}
			for k := 0; k < len(condition); k++ {
				if bytes.Compare(condition[k][:], tt.filter[j][k][:]) != 0 {
					t.Errorf("test %d, condition %d, segment %d: filter mismatch: have 0x%x, want 0x%x", i, j, k, condition[k], tt.filter[j][k])
				}
			}
		}
		// Check the binary creation
		binary := make([][][]byte, len(tt.topics))
		for j, condition := range tt.topics {
			binary[j] = make([][]byte, len(condition))
			for k, segment := range condition {
				binary[j][k] = []byte(segment)
			}
		}
		filter = NewFilterTopics(binary...)
		if len(filter) != len(tt.topics) {
			t.Errorf("test %d: condition count mismatch: have %v, want %v", i, len(filter), len(tt.topics))
			continue
		}
		for j, condition := range filter {
			if len(condition) != len(tt.filter[j]) {
				t.Errorf("test %d, condition %d: size mismatch: have %v, want %v", i, j, len(condition), len(tt.filter[j]))
				continue
			}
			for k := 0; k < len(condition); k++ {
				if bytes.Compare(condition[k][:], tt.filter[j][k][:]) != 0 {
					t.Errorf("test %d, condition %d, segment %d: filter mismatch: have 0x%x, want 0x%x", i, j, k, condition[k], tt.filter[j][k])
				}
			}
		}
	}
	// Check flat filter creation
	for i, tt := range filterTopicsCreationFlatTests {
		// Check the textual creation
		filter := NewFilterTopicsFromStringsFlat(tt.topics...)
		if len(filter) != len(tt.topics) {
			t.Errorf("test %d: condition count mismatch: have %v, want %v", i, len(filter), len(tt.topics))
			continue
		}
		for j, condition := range filter {
			if len(condition) != len(tt.filter[j]) {
				t.Errorf("test %d, condition %d: size mismatch: have %v, want %v", i, j, len(condition), len(tt.filter[j]))
				continue
			}
			for k := 0; k < len(condition); k++ {
				if bytes.Compare(condition[k][:], tt.filter[j][k][:]) != 0 {
					t.Errorf("test %d, condition %d, segment %d: filter mismatch: have 0x%x, want 0x%x", i, j, k, condition[k], tt.filter[j][k])
				}
			}
		}
		// Check the binary creation
		binary := make([][]byte, len(tt.topics))
		for j, topic := range tt.topics {
			binary[j] = []byte(topic)
		}
		filter = NewFilterTopicsFlat(binary...)
		if len(filter) != len(tt.topics) {
			t.Errorf("test %d: condition count mismatch: have %v, want %v", i, len(filter), len(tt.topics))
			continue
		}
		for j, condition := range filter {
			if len(condition) != len(tt.filter[j]) {
				t.Errorf("test %d, condition %d: size mismatch: have %v, want %v", i, j, len(condition), len(tt.filter[j]))
				continue
			}
			for k := 0; k < len(condition); k++ {
				if bytes.Compare(condition[k][:], tt.filter[j][k][:]) != 0 {
					t.Errorf("test %d, condition %d, segment %d: filter mismatch: have 0x%x, want 0x%x", i, j, k, condition[k], tt.filter[j][k])
				}
			}
		}
	}
}
