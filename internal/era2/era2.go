package era

import (
	"github.com/ethereum/go-ethereum/common"
)

type BlockProofHistoricalHashesAccumulator [15]common.Hash // 15 * 32 = 480 bytes

// BlockProofHistoricalRoots – Altair / Bellatrix historical_roots branch.
type BlockProofHistoricalRoots struct {
	BeaconBlockProof    [14]common.Hash // 448
	BeaconBlockRoot     common.Hash     // 32
	ExecutionBlockProof [11]common.Hash // 352
	Slot                uint64          // 8  => 840 bytes
}

// BlockProofHistoricalSummariesCapella – Capella historical_summaries branch.
type BlockProofHistoricalSummariesCapella struct {
	BeaconBlockProof    [13]common.Hash // 416
	BeaconBlockRoot     common.Hash     // 32
	ExecutionBlockProof [11]common.Hash // 352
	Slot                uint64          // 8  => 808 bytes
}

// BlockProofHistoricalSummariesDeneb – Deneb historical_summaries branch.
type BlockProofHistoricalSummariesDeneb struct {
	BeaconBlockProof    [13]common.Hash // 416
	BeaconBlockRoot     common.Hash     // 32
	ExecutionBlockProof [12]common.Hash // 384
	Slot                uint64          // 8  => 840 bytes
}
