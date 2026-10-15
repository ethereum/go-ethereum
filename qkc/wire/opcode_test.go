// Copyright 2026-2027, QuarkChain.

package wire

import "testing"

func TestOpcodeRangeInvariant(t *testing.T) {
	for i := 0; i < 256; i++ {
		b := byte(i)

		cmd := InCommandOpRange(b)
		clu := InClusterOpRange(b)
		if cmd && clu {
			t.Fatalf("opcode overlap at 0x%02X", b)
		}
		if !cmd && !clu {
			t.Fatalf("opcode not classified at 0x%02X", b)
		}
	}
}

func TestOpcodeBoundaryValues(t *testing.T) {
	tests := []struct {
		op        byte
		isCommand bool
		isCluster bool
	}{
		{0x00, true, false},
		{0x7F, true, false},
		{0x80, false, true},
		{0xFF, false, true},
	}

	for _, tt := range tests {
		if InCommandOpRange(tt.op) != tt.isCommand {
			t.Fatalf("command range mismatch at 0x%02X", tt.op)
		}
		if InClusterOpRange(tt.op) != tt.isCluster {
			t.Fatalf("cluster range mismatch at 0x%02X", tt.op)
		}
	}
}

func TestClusterOpBase(t *testing.T) {
	if ClusterOpBase != ClusterOp(128) {
		t.Errorf("ClusterOpBase = %d; want 128", ClusterOpBase)
	}
}

func TestCriticalOpcodeValues(t *testing.T) {
	cases := []struct {
		name string
		op   byte
	}{
		{"ClusterOpPing", byte(ClusterOpPing)},
		{"ClusterOpPong", byte(ClusterOpPong)},
		{"CommandOpHello", byte(CommandOpHello)},
		{"CommandOpPing", byte(CommandOpPing)},
		{"CommandOpPong", byte(CommandOpPong)},
	}

	for _, c := range cases {
		// range sanity
		if InCommandOpRange(c.op) == InClusterOpRange(c.op) {
			t.Fatalf("%s invalid classification", c.name)
		}
	}
}
