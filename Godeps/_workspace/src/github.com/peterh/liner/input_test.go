// +build !windows

package liner

import (
	"bufio"
	"bytes"
	"testing"
)

func (s *State) expectRune(t *testing.T, r rune) {
	item, err := s.readNext()
	if err != nil {
		t.Fatalf("Expected rune '%c', got error %s\n", r, err)
	}
	if v, ok := item.(rune); !ok {
		t.Fatalf("Expected rune '%c', got non-rune %v\n", r, v)
	} else {
		if v != r {
			t.Fatalf("Expected rune '%c', got rune '%c'\n", r, v)
		}
	}
}

func (s *State) expectAction(t *testing.T, a action) {
	item, err := s.readNext()
	if err != nil {
		t.Fatalf("Expected Action %d, got error %s\n", a, err)
	}
	if v, ok := item.(action); !ok {
		t.Fatalf("Expected Action %d, got non-Action %v\n", a, v)
	} else {
		if v != a {
			t.Fatalf("Expected Action %d, got Action %d\n", a, v)
		}
	}
}

func TestTypes(t *testing.T) {
	input := []byte{'A', 27, 'B', 27, 91, 68, 27, '[', '1', ';', '5', 'D', 'e'}
	var s State
	s.r = bufio.NewReader(bytes.NewBuffer(input))

	next := make(chan nexter)
	go func() {
		for {
			var n nexter
			n.r, _, n.err = s.r.ReadRune()
			next <- n
		}
	}()
	s.next = next

	s.expectRune(t, 'A')
	s.expectRune(t, 27)
	s.expectRune(t, 'B')
	s.expectAction(t, left)
	s.expectAction(t, wordLeft)

	s.expectRune(t, 'e')
}
