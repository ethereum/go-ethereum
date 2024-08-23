package types

import "slices"

//go:generate gencodec -type SubCircuitRowUsage -out gen_row_consumption_json.go
type SubCircuitRowUsage struct {
	Name      string `json:"name" gencodec:"required"`
	RowNumber uint64 `json:"row_number" gencodec:"required"`
}

// RowConsumptionLimit is the max number of row we support per subcircuit
// the actual limit is 1M but for safety we go with 950k
const RowConsumptionLimit = 950_000

type RowConsumption []SubCircuitRowUsage

// IsOverflown returns if any subcircuits are overflown
func (rc RowConsumption) IsOverflown() bool {
	return slices.ContainsFunc(rc, func(scru SubCircuitRowUsage) bool {
		return scru.RowNumber > RowConsumptionLimit
	})
}

// Difference returns rc - other
// Assumes that rc > other for all subcircuits
func (rc RowConsumption) Difference(other RowConsumption) RowConsumption {
	subCircuitMap := make(map[string]uint64, len(rc))
	for _, detail := range rc {
		subCircuitMap[detail.Name] = detail.RowNumber
	}

	for _, detail := range other {
		subCircuitMap[detail.Name] -= detail.RowNumber
	}

	diff := make([]SubCircuitRowUsage, 0, len(subCircuitMap))
	for name, rowNumDiff := range subCircuitMap {
		if rowNumDiff > 0 {
			diff = append(diff, SubCircuitRowUsage{
				Name:      name,
				RowNumber: rowNumDiff,
			})
		}
	}
	return diff
}
