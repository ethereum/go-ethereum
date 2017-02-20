package main

import "testing"

func lexAll(src string) []item {
	ch := lex("test.asm", []byte(src), false)

	var tokens []item
	for i := range ch {
		tokens = append(tokens, i)
	}
	return tokens
}

func TestComment(t *testing.T) {
	tokens := lexAll(";; this is a comment")
	if len(tokens) != 2 { // {new line, EOF}
		t.Error("expected no tokens")
	}
}
