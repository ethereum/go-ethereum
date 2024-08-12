package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRowConsumptionDifference(t *testing.T) {
	tests := []struct {
		rc1      RowConsumption
		rc2      RowConsumption
		expected RowConsumption
	}{
		{
			rc1: RowConsumption{
				SubCircuitRowUsage{
					"sc1",
					123,
				},
				SubCircuitRowUsage{
					"sc2",
					456,
				},
			},
			rc2: RowConsumption{
				SubCircuitRowUsage{
					"sc2",
					111,
				},
			},
			expected: RowConsumption{
				SubCircuitRowUsage{
					"sc1",
					123,
				},
				SubCircuitRowUsage{
					"sc2",
					345,
				},
			},
		},
		{
			rc1: RowConsumption{
				SubCircuitRowUsage{
					"sc1",
					123,
				},
				SubCircuitRowUsage{
					"sc2",
					456,
				},
			},
			rc2: RowConsumption{
				SubCircuitRowUsage{
					"sc2",
					456,
				},
			},
			expected: RowConsumption{
				SubCircuitRowUsage{
					"sc1",
					123,
				},
			},
		},
	}

	makeMap := func(rc RowConsumption) map[string]uint64 {
		m := make(map[string]uint64)
		for _, usage := range rc {
			m[usage.Name] = usage.RowNumber
		}
		return m
	}

	for _, test := range tests {
		assert.Equal(t, makeMap(test.expected), makeMap(test.rc1.Difference(test.rc2)))
	}
}
