package ethtest

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

type Chain struct {
	blocks      []*types.Block
	chainConfig *params.ChainConfig
}

func (c *Chain) WriteTo(writer io.Writer) error {
	for _, block := range c.blocks {
		if err := rlp.Encode(writer, block); err != nil {
			return err
		}
	}

	return nil
}

// Len returns the length of the chain.
func (c *Chain) Len() int {
	return len(c.blocks)
}

// TD calculates the total difficulty of the chain.
func (c *Chain) TD(height int) *big.Int { // TODO later on channge scheme so that the height is included in range
	sum := big.NewInt(0)
	for _, block := range c.blocks[:height] {
		sum.Add(sum, block.Difficulty())
	}
	return sum
}

// ForkID gets the fork id of the chain.
func (c *Chain) ForkID() forkid.ID {
	return forkid.NewID(c.chainConfig, c.blocks[0].Hash(), uint64(c.Len()))
}

// Shorten returns a copy chain of a desired height from the imported
func (c *Chain) Shorten(height int) *Chain {
	blocks := make([]*types.Block, height)
	copy(blocks, c.blocks[:height])

	config := *c.chainConfig
	return &Chain{
		blocks:      blocks,
		chainConfig: &config,
	}
}

// Head returns the chain head.
func (c *Chain) Head() *types.Block {
	return c.blocks[c.Len()-1]
}

// loadChain takes the given chain.rlp file, and decodes and returns
// the blocks from the file.
func loadChain(chainfile string, genesis string) (*Chain, error) {
	// Open the file handle and potentially unwrap the gzip stream
	fh, err := os.Open(chainfile)
	if err != nil {
		return nil, err
	}
	defer fh.Close()

	var reader io.Reader = fh
	if strings.HasSuffix(chainfile, ".gz") {
		if reader, err = gzip.NewReader(reader); err != nil {
			return nil, err
		}
	}
	stream := rlp.NewStream(reader, 0)
	var blocks []*types.Block
	for i := 0; ; i++ {
		var b types.Block
		if err := stream.Decode(&b); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("at block %d: %v", i, err)
		}
		blocks = append(blocks, &b)
	}

	// Open the file handle and potentially unwrap the gzip stream
	chainConfig, err := ioutil.ReadFile(genesis)
	if err != nil {
		return nil, err
	}
	var gen core.Genesis
	if err := json.Unmarshal(chainConfig, &gen); err != nil {
		return nil, err
	}

	return &Chain{
		blocks:      blocks,
		chainConfig: gen.Config,
	}, nil
}
