package types

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	gokzg4844 "github.com/crate-crypto/go-kzg-4844"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto/kzg"
	"github.com/ethereum/go-ethereum/params"

	"github.com/protolambda/ztyp/codec"
)

// Compressed BLS12-381 G1 element
type KZGCommitment [48]byte

func (p *KZGCommitment) Deserialize(dr *codec.DecodingReader) error {
	if p == nil {
		return errors.New("nil pubkey")
	}
	_, err := dr.Read(p[:])
	return err
}

func (p *KZGCommitment) Serialize(w *codec.EncodingWriter) error {
	return w.Write(p[:])
}

func (KZGCommitment) ByteLength() uint64 {
	return 48
}

func (KZGCommitment) FixedLength() uint64 {
	return 48
}

func (p KZGCommitment) MarshalText() ([]byte, error) {
	return []byte("0x" + hex.EncodeToString(p[:])), nil
}

func (p KZGCommitment) String() string {
	return "0x" + hex.EncodeToString(p[:])
}

func (p *KZGCommitment) UnmarshalText(text []byte) error {
	return hexutil.UnmarshalFixedText("KZGCommitment", text, p[:])
}

func (c KZGCommitment) ComputeVersionedHash() common.Hash {
	return common.Hash(kzg.KZGToVersionedHash(gokzg4844.KZGCommitment(c)))
}

// Compressed BLS12-381 G1 element
type KZGProof [48]byte

func (p *KZGProof) Deserialize(dr *codec.DecodingReader) error {
	if p == nil {
		return errors.New("nil pubkey")
	}
	_, err := dr.Read(p[:])
	return err
}

func (p *KZGProof) Serialize(w *codec.EncodingWriter) error {
	return w.Write(p[:])
}

func (KZGProof) ByteLength() uint64 {
	return 48
}

func (KZGProof) FixedLength() uint64 {
	return 48
}

func (p KZGProof) MarshalText() ([]byte, error) {
	return []byte("0x" + hex.EncodeToString(p[:])), nil
}

func (p KZGProof) String() string {
	return "0x" + hex.EncodeToString(p[:])
}

func (p *KZGProof) UnmarshalText(text []byte) error {
	return hexutil.UnmarshalFixedText("KZGProof", text, p[:])
}

// BLSFieldElement is the raw bytes representation of a field element
type BLSFieldElement [32]byte

func (p BLSFieldElement) MarshalText() ([]byte, error) {
	return []byte("0x" + hex.EncodeToString(p[:])), nil
}

func (p BLSFieldElement) String() string {
	return "0x" + hex.EncodeToString(p[:])
}

func (p *BLSFieldElement) UnmarshalText(text []byte) error {
	return hexutil.UnmarshalFixedText("BLSFieldElement", text, p[:])
}

// Blob data
type Blob [params.FieldElementsPerBlob * 32]byte

func (blob *Blob) Deserialize(dr *codec.DecodingReader) error {
	if blob == nil {
		return errors.New("cannot decode ssz into nil Blob")
	}

	// We treat the blob as an opaque sequence of bytes
	// and therefore we do not do any validation related to field
	// elements
	if _, err := dr.Read(blob[:]); err != nil {
		return err
	}

	return nil
}

func (blob *Blob) Serialize(w *codec.EncodingWriter) error {
	return w.Write(blob[:])

}

func (blob *Blob) ByteLength() (out uint64) {
	return params.FieldElementsPerBlob * 32
}

func (blob *Blob) FixedLength() uint64 {
	return params.FieldElementsPerBlob * 32
}

func (blob *Blob) MarshalText() ([]byte, error) {
	out := make([]byte, 2+params.FieldElementsPerBlob*32*2)
	copy(out[:2], "0x")
	hex.Encode(out[2:], blob[:])

	return out, nil
}

func (blob *Blob) String() string {
	v, err := blob.MarshalText()
	if err != nil {
		return "<invalid-blob>"
	}
	return string(v)
}

func (blob *Blob) UnmarshalText(text []byte) error {
	if blob == nil {
		return errors.New("cannot decode text into nil Blob")
	}
	l := 2 + params.FieldElementsPerBlob*32*2
	if len(text) != l {
		return fmt.Errorf("expected %d characters but got %d", l, len(text))
	}
	if !(text[0] == '0' && text[1] == 'x') {
		return fmt.Errorf("expected '0x' prefix in Blob string")
	}
	if _, err := hex.Decode(blob[:], text[2:]); err != nil {
		return fmt.Errorf("blob is not formatted correctly: %v", err)
	}

	return nil
}

type BlobKzgs []KZGCommitment

func (li *BlobKzgs) Deserialize(dr *codec.DecodingReader) error {
	return dr.List(func() codec.Deserializable {
		i := len(*li)
		*li = append(*li, KZGCommitment{})
		return &(*li)[i]
	}, 48, params.MaxBlobsPerBlock)
}

func (li BlobKzgs) Serialize(w *codec.EncodingWriter) error {
	return w.List(func(i uint64) codec.Serializable {
		return &li[i]
	}, 48, uint64(len(li)))
}

func (li BlobKzgs) ByteLength() uint64 {
	return uint64(len(li)) * 48
}

func (li BlobKzgs) FixedLength() uint64 {
	return 0
}

func (li BlobKzgs) copy() BlobKzgs {
	cpy := make(BlobKzgs, len(li))
	copy(cpy, li)
	return cpy
}

type KZGProofs []KZGProof

func (li *KZGProofs) Deserialize(dr *codec.DecodingReader) error {
	return dr.List(func() codec.Deserializable {
		i := len(*li)
		*li = append(*li, KZGProof{})
		return &(*li)[i]
	}, 48, params.MaxBlobsPerBlock)
}

func (li KZGProofs) Serialize(w *codec.EncodingWriter) error {
	return w.List(func(i uint64) codec.Serializable {
		return &li[i]
	}, 48, uint64(len(li)))
}

func (li KZGProofs) ByteLength() uint64 {
	return uint64(len(li)) * 48
}

func (li KZGProofs) FixedLength() uint64 {
	return 0
}

func (li KZGProofs) copy() KZGProofs {
	cpy := make(KZGProofs, len(li))
	copy(cpy, li)
	return cpy
}

type Blobs []Blob

func (a *Blobs) Deserialize(dr *codec.DecodingReader) error {
	return dr.List(func() codec.Deserializable {
		i := len(*a)
		*a = append(*a, Blob{})
		return &(*a)[i]
	}, params.FieldElementsPerBlob*32, params.FieldElementsPerBlob)
}

func (a Blobs) Serialize(w *codec.EncodingWriter) error {
	return w.List(func(i uint64) codec.Serializable {
		return &a[i]
	}, params.FieldElementsPerBlob*32, uint64(len(a)))
}

func (a Blobs) ByteLength() (out uint64) {
	return uint64(len(a)) * params.FieldElementsPerBlob * 32
}

func (a *Blobs) FixedLength() uint64 {
	return 0 // it's a list, no fixed length
}

func (blobs Blobs) copy() Blobs {
	cpy := make(Blobs, len(blobs))
	copy(cpy, blobs) // each blob element is an array and gets deep-copied
	return cpy
}

// Return KZG commitments, versioned hashes and the proofs that correspond to these blobs
func (blobs Blobs) ComputeCommitmentsAndProofs() (commitments []KZGCommitment, versionedHashes []common.Hash, proofs []KZGProof, err error) {
	commitments = make([]KZGCommitment, len(blobs))
	proofs = make([]KZGProof, len(blobs))
	versionedHashes = make([]common.Hash, len(blobs))

	cryptoCtx := kzg.CrpytoCtx()
	for i, blob := range blobs {
		commitment, err := cryptoCtx.BlobToKZGCommitment(gokzg4844.Blob(blob))
		if err != nil {
			return nil, nil, nil, fmt.Errorf("could not convert blob to commitment: %v", err)
		}

		proof, err := cryptoCtx.ComputeBlobKZGProof(gokzg4844.Blob(blob), commitment)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("could not compute proof for blob: %v", err)
		}
		commitments[i] = KZGCommitment(commitment)
		proofs[i] = KZGProof(proof)
		versionedHashes[i] = common.Hash(kzg.KZGToVersionedHash(gokzg4844.KZGCommitment(commitment)))
	}

	return commitments, versionedHashes, proofs, nil
}

func toBlobs(_blobs Blobs) []gokzg4844.Blob {
	blobs := make([]gokzg4844.Blob, len(_blobs))
	for i, _blob := range _blobs {
		blobs[i] = gokzg4844.Blob(_blob)
	}
	return blobs
}
func toComms(_comms BlobKzgs) []gokzg4844.KZGCommitment {
	comms := make([]gokzg4844.KZGCommitment, len(_comms))
	for i, _comm := range _comms {
		comms[i] = gokzg4844.KZGCommitment(_comm)
	}
	return comms
}
func toProofs(_proofs KZGProofs) []gokzg4844.KZGProof {
	proofs := make([]gokzg4844.KZGProof, len(_proofs))
	for i, _proof := range _proofs {
		proofs[i] = gokzg4844.KZGProof(_proof)
	}
	return proofs
}

type BlobTxWrapper struct {
	Tx       SignedBlobTx
	BlobKzgs BlobKzgs
	Blobs    Blobs
	Proofs   KZGProofs
}

func (txw *BlobTxWrapper) Deserialize(dr *codec.DecodingReader) error {
	return dr.Container(&txw.Tx, &txw.BlobKzgs, &txw.Blobs, &txw.Proofs)
}

func (txw *BlobTxWrapper) Serialize(w *codec.EncodingWriter) error {
	return w.Container(&txw.Tx, &txw.BlobKzgs, &txw.Blobs, &txw.Proofs)
}

func (txw *BlobTxWrapper) ByteLength() uint64 {
	return codec.ContainerLength(&txw.Tx, &txw.BlobKzgs, &txw.Blobs, &txw.Proofs)
}

func (txw *BlobTxWrapper) FixedLength() uint64 {
	return 0
}

type BlobTxWrapData struct {
	BlobKzgs BlobKzgs
	Blobs    Blobs
	Proofs   KZGProofs
}

// sizeWrapData returns the size in bytes of the ssz-encoded BlobTxWrapData
func (b *BlobTxWrapData) sizeWrapData() common.StorageSize {
	return common.StorageSize(codec.ContainerLength(&b.BlobKzgs, &b.Blobs, &b.Proofs))
}

// validateBlobTransactionWrapper implements validate_blob_transaction_wrapper from EIP-4844
func (b *BlobTxWrapData) validateBlobTransactionWrapper(inner TxData) error {
	blobTx, ok := inner.(*SignedBlobTx)
	if !ok {
		return fmt.Errorf("expected signed blob tx, got %T", inner)
	}
	l1 := len(b.BlobKzgs)
	l2 := len(blobTx.Message.BlobVersionedHashes)
	l3 := len(b.Blobs)
	l4 := len(b.Proofs)
	if l1 != l2 || l1 != l3 || l1 != l4 {
		return fmt.Errorf("lengths don't match %v %v %v %v", l1, l2, l3, l4)
	}
	// the following check isn't strictly necessary as it would be caught by data gas processing
	// (and hence it is not explicitly in the spec for this function), but it doesn't hurt to fail
	// early in case we are getting spammed with too many blobs or there is a bug somewhere:
	if l1 > params.MaxBlobsPerBlock {
		return fmt.Errorf("number of blobs exceeds max: %v", l1)
	}
	cryptoCtx := kzg.CrpytoCtx()
	err := cryptoCtx.VerifyBlobKZGProofBatch(toBlobs(b.Blobs), toComms(b.BlobKzgs), toProofs(b.Proofs))
	if err != nil {
		return fmt.Errorf("error during proof verification: %v", err)
	}
	if !ok {
		return errors.New("failed to verify kzg")
	}
	for i, h := range blobTx.Message.BlobVersionedHashes {
		if computed := b.BlobKzgs[i].ComputeVersionedHash(); computed != h {
			return fmt.Errorf("versioned hash %d supposedly %s but does not match computed %s", i, h, computed)
		}
	}
	return nil
}

func (b *BlobTxWrapData) copy() TxWrapData {
	return &BlobTxWrapData{
		BlobKzgs: b.BlobKzgs.copy(),
		Blobs:    b.Blobs.copy(),
		Proofs:   b.Proofs.copy(),
	}
}

func (b *BlobTxWrapData) kzgs() BlobKzgs {
	return b.BlobKzgs
}

func (b *BlobTxWrapData) blobs() Blobs {
	return b.Blobs
}

func (b *BlobTxWrapData) proofs() KZGProofs {
	return b.Proofs
}

func (b *BlobTxWrapData) encodeTyped(w io.Writer, txdata TxData) error {
	if _, err := w.Write([]byte{BlobTxType}); err != nil {
		return err
	}
	blobTx, ok := txdata.(*SignedBlobTx)
	if !ok {
		return fmt.Errorf("expected signed blob tx, got %T", txdata)
	}
	wrapped := BlobTxWrapper{
		Tx:       *blobTx,
		BlobKzgs: b.BlobKzgs,
		Blobs:    b.Blobs,
		Proofs:   b.Proofs,
	}
	return EncodeSSZ(w, &wrapped)
}
