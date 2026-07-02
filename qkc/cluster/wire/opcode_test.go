// Copyright 2026-2027, QuarkChain.

package wire

import (
	"strings"
	"testing"
)

func TestOpcodeClassificationAndBoundaries(t *testing.T) {
	tests := []struct {
		op        byte
		isCommand bool
		isCluster bool
	}{
		{0, true, false},
		{127, true, false},
		{128, false, true},
		{255, false, true},
	}

	for _, tt := range tests {
		if got := InCommandOpRange(tt.op); got != tt.isCommand {
			t.Errorf("IsCommandOp(%d) = %v; want %v", tt.op, got, tt.isCommand)
		}
		if got := InClusterOpRange(tt.op); got != tt.isCluster {
			t.Errorf("IsClusterOp(%d) = %v; want %v", tt.op, got, tt.isCluster)
		}
	}
}

func TestClusterErrorFormatting(t *testing.T) {
	tests := []struct {
		err          *ClusterError
		wantContains []string
	}{
		{
			err:          NewClusterError(404, "not found"),
			wantContains: []string{"404", "not found"},
		},
		{
			err:          NewClusterError(500, ""),
			wantContains: []string{"500", "cluster error"},
		},
	}

	for _, tt := range tests {
		got := tt.err.Error()
		for _, want := range tt.wantContains {
			if !strings.Contains(got, want) {
				t.Errorf("ClusterError.Error() = %q; missing %q", got, want)
			}
		}
	}
}
