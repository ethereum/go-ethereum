package tracers

import (
	"testing"

	pbeth "github.com/streamingfast/firehose-ethereum/types/pb/sf/ethereum/type/v2"
	"github.com/stretchr/testify/require"
)

func TestFirehoseCallStack_Push(t *testing.T) {
	type actionRunner func(t *testing.T, s *CallStack)

	push := func(call *pbeth.Call) actionRunner { return func(_ *testing.T, s *CallStack) { s.Push(call) } }
	pop := func() actionRunner { return func(_ *testing.T, s *CallStack) { s.Pop() } }
	check := func(r actionRunner) actionRunner { return func(t *testing.T, s *CallStack) { r(t, s) } }

	tests := []struct {
		name    string
		actions []actionRunner
	}{
		{
			"push/pop emtpy", []actionRunner{
				push(&pbeth.Call{}),
				pop(),
				check(func(t *testing.T, s *CallStack) {
					require.Len(t, s.stack, 0)
				}),
			},
		},
		{
			"push/push/push", []actionRunner{
				push(&pbeth.Call{}),
				push(&pbeth.Call{}),
				push(&pbeth.Call{}),
				check(func(t *testing.T, s *CallStack) {
					require.Len(t, s.stack, 3)

					require.Equal(t, 1, int(s.stack[0].Index))
					require.Equal(t, 0, int(s.stack[0].ParentIndex))

					require.Equal(t, 2, int(s.stack[1].Index))
					require.Equal(t, 1, int(s.stack[1].ParentIndex))

					require.Equal(t, 3, int(s.stack[2].Index))
					require.Equal(t, 2, int(s.stack[2].ParentIndex))
				}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewCallStack()

			for _, action := range tt.actions {
				action(t, s)
			}
		})
	}
}

func Test_validateKnownTransactionTypes(t *testing.T) {
	tests := []struct {
		name      string
		txType    byte
		knownType bool
		want      error
	}{
		{"legacy", 0, true, nil},
		{"access_list", 1, true, nil},
		{"inexistant", 255, false, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFirehoseKnownTransactionType(tt.txType, tt.knownType)
			if tt.want == nil && err != nil {
				t.Fatalf("Transaction of type %d expected to validate properly but received error %q", tt.txType, err)
			} else if tt.want != nil && err == nil {
				t.Fatalf("Transaction of type %d expected to validate improperly but generated no error", tt.txType)
			} else if tt.want != nil && err != nil && tt.want.Error() != err.Error() {
				t.Fatalf("Transaction of type %d expected to validate improperly but generated error %q does not match expected error %q", tt.txType, err, tt.want)
			}
		})
	}
}
