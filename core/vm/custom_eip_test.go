package vm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtendActivators(t *testing.T) {
	eips_snapshot := GetActivatorsEipNumbers()

	testCases := []struct {
		name          string
		newActivators map[int]func(*JumpTable)
		expPass       bool
		errContains   string
		postCheck     func()
	}{
		{
			"success - nil new activators",
			nil,
			true,
			"",
			func() {
				eips := GetActivatorsEipNumbers()
				require.ElementsMatch(t, eips_snapshot, eips, "expected eips number to be equal")
			},
		},
		{
			"success - single new activator",
			map[int]func(*JumpTable){
				0o000: func(jt *JumpTable) {},
			},
			true,
			"",
			func() {
				eips := GetActivatorsEipNumbers()
				require.ElementsMatch(t, append(eips_snapshot, 0), eips, "expected eips number to be equal")
			},
		},
		{
			"success - multiple new activators",
			map[int]func(*JumpTable){
				0o001: func(jt *JumpTable) {},
				0o002: func(jt *JumpTable) {},
			},
			true,
			"",
			func() {
				eips := GetActivatorsEipNumbers()
				// since we are working with a global function, tests are not independent
				require.ElementsMatch(t, append(eips_snapshot, 0, 1, 2), eips,
					"expected eips number to be equal")
			},
		},
		{
			"fail - repeated activator",
			map[int]func(*JumpTable){
				3855: func(jt *JumpTable) {},
			},
			false,
			"",
			func() {
				eips := GetActivatorsEipNumbers()
				// since we are working with a global function, tests are not independent
				require.ElementsMatch(t, append(eips_snapshot, 0, 1, 2), eips,
					"expected eips number to be equal")
			},
		},
		{
			"fail - valid activator is not stored if a repeated is present",
			map[int]func(*JumpTable){
				0o003: func(jt *JumpTable) {},
				3855:  func(jt *JumpTable) {},
			},
			false,
			"",
			func() {
				eips := GetActivatorsEipNumbers()
				// since we are working with a global function, tests are not independent
				require.ElementsMatch(t, append(eips_snapshot, 0o000, 0o001, 0o002), eips,
					"expected eips number to be equal")
			},
		},
	}

	for _, tc := range testCases {
		err := ExtendActivators(tc.newActivators)
		if tc.expPass {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.errContains, "expected different error")
		}

		tc.postCheck()
	}
}

func TestAddOperation(t *testing.T) {
	// Functions used to create an operation.
	customExecute := func(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
		// no - op
		return nil, nil
	}
	customDynamicGas := func(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (uint64, error) {
		// no-op
		return 0, nil
	}
	customMemorySize := func(stack *Stack) (uint64, bool) {
		// no-op
		return 0, false
	}

	const (
		EXISTENT OpCode = STOP
		NEW      OpCode = 0xf
	)

	testCases := []struct {
		name        string
		opName      string
		opNumber    OpCode
		expPass     bool
		errContains string
		postCheck   func()
	}{
		{
			"fail - operation with same number already exists",
			"TEST",
			EXISTENT,
			false,
			"already exists",
			func() {
				name := EXISTENT.String()
				require.Equal(t, "STOP", name)
			},
		},
		{
			"fail - operation with same name already exists",
			"CREATE",
			NEW,
			false,
			"already exists",
			func() {
				name := NEW.String()
				require.Contains(t, name, "not defined")
			},
		},
		{
			"fail - operation with same name of STOP",
			"STOP",
			NEW,
			false,
			"already exists",
			func() {
				name := NEW.String()
				require.Contains(t, name, "not defined")
			},
		},
		{
			"pass - new operation added to the list",
			"TEST",
			NEW,
			true,
			"",
			func() {
				name := NEW.String()
				require.Equal(t, "TEST", name)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opInfo := OpCodeInfo{
				Number: tc.opNumber,
				Name:   tc.opName,
			}
			_, err := ExtendOperations(opInfo, customExecute, 0, customDynamicGas, 0, 0, customMemorySize)

			if tc.expPass {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains, "expected different error")
			}

			tc.postCheck()
		})
	}
}
