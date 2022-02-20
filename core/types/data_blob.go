package types

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/protolambda/ztyp/codec"
	"github.com/protolambda/ztyp/tree"
	"io"
)

// Compressed G1 element
type KZGCommitment [48]byte

func (kzg KZGCommitment) ComputeVersionedHash() common.Hash {
	return common.Hash{} // TODO George
}

// Blob data
type Blob [][32]byte

func (bl Blob) copy() Blob {
	cpy := make(Blob, len(bl))
	copy(cpy, bl)
	return cpy
}

func (bl Blob) ComputeCommitment() KZGCommitment {
	return KZGCommitment{} // TODO George
}

type BlobKzgs []KZGCommitment

func (kzgs BlobKzgs) copy() BlobKzgs {
	cpy := make(BlobKzgs, len(kzgs))
	copy(cpy, kzgs)
	return cpy
}

type Blobs []Blob

func (blobs Blobs) copy() Blobs {
	cpy := make(Blobs, len(blobs))
	for i, bl := range blobs {
		cpy[i] = bl.copy()
	}
	return cpy
}

type BlobTxWrapper struct {
	Tx       SignedBlobTx
	BlobKzgs BlobKzgs
	Blobs    Blobs
}

func (txw *BlobTxWrapper) Deserialize(dr *codec.DecodingReader) error {
	return dr.Container(&txw.Tx, &txw.BlobKzgs, &txw.Blobs)
}

func (txw *BlobTxWrapper) Serialize(w *codec.EncodingWriter) error {
	return w.Container(&txw.Tx, &txw.BlobKzgs, &txw.Blobs)
}

func (txw *BlobTxWrapper) ByteLength() uint64 {
	return codec.ContainerLength(&txw.Tx, &txw.BlobKzgs, &txw.Blobs)
}

func (txw *BlobTxWrapper) FixedLength() uint64 {
	return 0
}

func (txw *BlobTxWrapper) HashTreeRoot(hFn tree.HashFn) tree.Root {
	return hFn.HashTreeRoot(&txw.Tx, &txw.BlobKzgs, &txw.Blobs)
}

type BlobTxWrapData struct {
	BlobKzgs BlobKzgs
	Blobs    Blobs
}

func (b *BlobTxWrapData) sizeWrapData() common.StorageSize {
	return common.StorageSize(4 + 4 + b.BlobKzgs.ByteLength() + b.Blobs.ByteLength())
}

func (b *BlobTxWrapData) checkWrapping(inner TxData) error {
	// TODO george: check if tx data matches wrap contents. versioned hashes <> kzgs <> blobs
	return nil
}

func (b *BlobTxWrapData) copy() TxWrapData {
	return &BlobTxWrapData{
		BlobKzgs: b.BlobKzgs.copy(),
		Blobs:    b.Blobs.copy(),
	}
}

func (b *BlobTxWrapData) kzgs() BlobKzgs {
	return b.BlobKzgs
}

func (b *BlobTxWrapData) blobs() Blobs {
	return b.Blobs
}

func (b *BlobTxWrapData) encodeTyped(w io.Writer, txdata TxData) error {
	blobTx, ok := txdata.(*SignedBlobTx)
	if !ok {
		return fmt.Errorf("expected signed blob tx, got %T", txdata)
	}
	wrapped := BlobTxWrapper{
		Tx:       *blobTx,
		BlobKzgs: b.BlobKzgs,
		Blobs:    b.Blobs,
	}
	return EncodeSSZ(w, &wrapped)
}
