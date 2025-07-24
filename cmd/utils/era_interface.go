package utils

import (
	"io"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/era"
	"github.com/ethereum/go-ethereum/internal/era2"
)

// Iterator is the common, read‑only view returned by both Era‑1 and Era‑E.
type Iterator interface {
	Next() bool
	Number() uint64
	Block() (*types.Block, error)
	Receipts() (types.Receipts, error)
	Error() error
}

// Builder (unchanged) – writes one archive file.
type Builder interface {
	Add(blk *types.Block, rcpts types.Receipts, td *big.Int, proof era2.Proof) error
	Finalize() (common.Hash, error)
}

// Format now encapsulates *all* format‑specific behaviour, both read & write.
type Format interface {
	// ----- writer side -----
	Filename(network string, epoch int, root common.Hash) string
	NewBuilder(w io.Writer) Builder

	// ----- reader side -----
	ReadDir(dir, network string) ([]string, error)
	NewIterator(f *os.File) (Iterator, error)
}

type era1Builder struct{ *era.Builder }

func (b *era1Builder) Add(blk *types.Block, rc types.Receipts, td *big.Int, proof era2.Proof) error {
	return b.Builder.Add(blk, rc, td)
}

type era1Format struct{}

func (era1Format) Filename(n string, e int, h common.Hash) string { return era.Filename(n, e, h) }
func (era1Format) NewBuilder(w io.Writer) Builder                 { return &era1Builder{era.NewBuilder(w)} }
func (era1Format) ReadDir(dir, net string) ([]string, error)      { return era.ReadDir(dir, net) }
func (era1Format) NewIterator(f *os.File) (Iterator, error) {
	e, err := era.From(f)
	if err != nil {
		return nil, err
	}
	return era.NewIterator(e)
}

var Era1 Format = era1Format{}

type eraeBuilder struct{ *era2.Builder }

func (b *eraeBuilder) Add(blk *types.Block, rc types.Receipts, td *big.Int, proof era2.Proof) error {
	return b.Builder.Add(blk, rc, td, nil) // no proofs yet
}

type eraeFormat struct{}

func (eraeFormat) Filename(n string, e int, h common.Hash) string { return era2.Filename(n, e, h) }
func (eraeFormat) NewBuilder(w io.Writer) Builder                 { return &eraeBuilder{era2.NewBuilder(w)} }
func (eraeFormat) ReadDir(dir, net string) ([]string, error)      { return era2.ReadDir(dir, net) }
func (eraeFormat) NewIterator(f *os.File) (Iterator, error) {
	e, err := era2.From(f)
	if err != nil {
		return nil, err
	}
	return era2.NewIterator(e)
}

var EraE Format = eraeFormat{}

// Exported singleton
