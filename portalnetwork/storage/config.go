package storage

import (
	"database/sql"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/protolambda/zrnt/eth2/beacon/common"
)

type PortalStorageConfig struct {
	StorageCapacityMB uint64
	DB                *sql.DB
	NodeId            enode.ID
	Spec              *common.Spec
}
