// nolint
package rawdb

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

var (
	lastCheckpoint = []byte("LastCheckpoint")

	ErrEmptyLastFinality                    = errors.New("empty response while getting last finality")
	ErrIncorrectFinality                    = errors.New("last checkpoint in the DB is incorrect")
	ErrIncorrectFinalityToStore             = errors.New("failed to marshal the last finality struct")
	ErrDBNotResponding                      = errors.New("failed to store the last finality struct")
	ErrIncorrectLockFieldToStore            = errors.New("failed to marshal the lockField struct ")
	ErrIncorrectLockField                   = errors.New("lock field in the DB is incorrect")
	ErrIncorrectFutureMilestoneFieldToStore = errors.New("failed to marshal the future milestone field struct ")
	ErrIncorrectFutureMilestoneField        = errors.New("future milestone field  in the DB is incorrect")
)

type Checkpoint struct {
	Finality
}

func (c *Checkpoint) clone() *Checkpoint {
	return &Checkpoint{}
}

func (c *Checkpoint) block() (uint64, common.Hash) {
	return c.Block, c.Hash
}
