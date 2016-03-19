package vm

import "testing"

func TestInit(t *testing.T) {
	if frontierJumpTable[DELEGATECALL].valid {
		t.Error("Expected DELEGATECALL not to be present")
	}

	if !homesteadJumpTable[DELEGATECALL].valid {
		t.Error("Expected DELEGATECALL not to be present")
	}
}
