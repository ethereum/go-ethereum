package eth

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/golang/snappy"
)

type RLPFileSetReplicator struct {
	filesetBasePath   string
	nodeStartID       uint64
	chunkSeq          uint64
	chunkFile         *os.File
	chunkBytesWritten uint64
	description       string
	compBuf           []byte
}

const rlpFileReplicationChunkSizeLimit = 5 * 1024 * 1024
const rlpFileReplicationCompBufSize = 20 * 1024 * 1024

func NewRLPFileSetReplicator(filesetBaseURI *url.URL) (*core.ChainReplicator, error) {
	filesetBasePath := filesetBaseURI.Path

	err := os.MkdirAll(filesetBasePath, 0755)
	if err != nil {
		return nil, err
	}

	nodeStartID := uint64(time.Now().Unix())

	backend := &RLPFileSetReplicator{
		filesetBasePath: filesetBasePath,
		nodeStartID:     nodeStartID,
		description:     fmt.Sprintf("RLPFileSet(path=%s)", filesetBasePath),
		compBuf:         make([]byte, 0, rlpFileReplicationCompBufSize),
	}

	err = backend.OpenNextChunk()
	if err != nil {
		return nil, err
	}

	return core.NewChainReplicator(backend), nil
}

func (r *RLPFileSetReplicator) OpenNextChunk() error {
	if r.chunkFile != nil {
		r.chunkFile.Close()
		r.chunkSeq++
	}

	chunkFileName := fmt.Sprintf("sess-%08x-chunk-%08x.rlp", r.nodeStartID, r.chunkSeq)
	chunkFilePath := path.Join(r.filesetBasePath, chunkFileName)

	f, err := os.Create(chunkFilePath)
	if err != nil {
		return err
	}

	r.chunkFile = f
	r.chunkBytesWritten = 0

	return nil
}

func (r *RLPFileSetReplicator) String() string {
	return r.description
}

func (r *RLPFileSetReplicator) Process(ctx context.Context, events []*core.BlockReplicationEvent) (err error) {
	var (
		rlpData              []byte
		compRlpData          []byte
		bytesWrittenForStep  uint64
		bytesWrittenForEvent int
	)

	for _, event := range events {
		rlpData, err = rlp.EncodeToBytes([]interface{}{
			event.Hash,
			event.Data,
		})
		if err != nil {
			return
		}

		compRlpData = snappy.Encode(r.compBuf, rlpData)

		bytesWrittenForEvent, err = r.chunkFile.Write(compRlpData)
		if err != nil {
			return
		}
		bytesWrittenForStep += uint64(bytesWrittenForEvent)
	}

	err = r.chunkFile.Sync()
	if err != nil {
		return
	}

	r.chunkBytesWritten += bytesWrittenForStep
	if r.chunkBytesWritten >= rlpFileReplicationChunkSizeLimit {
		err = r.OpenNextChunk()
	}

	return
}
