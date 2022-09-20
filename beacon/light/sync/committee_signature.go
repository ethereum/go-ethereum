// Copyright 2022 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package sync

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/beacon/light/types"
	"github.com/ethereum/go-ethereum/beacon/merkle"
	"github.com/ethereum/go-ethereum/beacon/params"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/minio/sha256-simd"
	bls "github.com/protolambda/bls12-381-util"
)

// syncCommittee holds either a blsSyncCommittee or a fake dummySyncCommittee used for testing
type syncCommittee interface{}

// committeeSigVerifier verifies sync committee signatures (either proper BLS
// signatures or fake signatures used for testing)
type committeeSigVerifier interface {
	deserializeSyncCommittee(enc []byte) syncCommittee
	verifySignature(committee syncCommittee, signedRoot common.Hash, bitmask, signature []byte) bool
}

// blsSyncCommittee is a set of sync committee signer pubkeys
//
// See data structure definition here:
// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/beacon-chain.md#syncaggregate
type blsSyncCommittee struct {
	keys      [params.SyncCommitteeSize]*bls.Pubkey
	aggregate *bls.Pubkey
}

// BLSVerifier implements committeeSigVerifier
type BLSVerifier struct{}

// deserializeSyncCommittee implements committeeSigVerifier
func (BLSVerifier) deserializeSyncCommittee(enc []byte) syncCommittee {
	if len(enc) != SerializedCommitteeSize {
		log.Error("Wrong input size for deserializeSyncCommittee", "expected", SerializedCommitteeSize, "got", len(enc))
		return nil
	}
	sc := new(blsSyncCommittee)
	for i := 0; i <= params.SyncCommitteeSize; i++ {
		pk := new(bls.Pubkey)
		var sk [params.BlsPubkeySize]byte
		copy(sk[:], enc[i*params.BlsPubkeySize:(i+1)*params.BlsPubkeySize])
		if err := pk.Deserialize(&sk); err != nil {
			log.Error("bls.Pubkey.Deserialize failed", "error", err, "data", sk)
			return nil
		}
		if i < params.SyncCommitteeSize {
			sc.keys[i] = pk
		} else {
			sc.aggregate = pk
		}
	}
	return sc
}

// verifySignature implements committeeSigVerifier
func (BLSVerifier) verifySignature(committee syncCommittee, signingRoot common.Hash, bitmask, signature []byte) bool {
	if len(signature) != params.BlsSignatureSize || len(bitmask) != params.SyncCommitteeSize/8 {
		return false
	}
	var (
		sig          bls.Signature
		sigBytes     [params.BlsSignatureSize]byte
		signerKeys   [params.SyncCommitteeSize]*bls.Pubkey
		signerCount  int
		blsCommittee = committee.(*blsSyncCommittee)
	)
	copy(sigBytes[:], signature)
	if err := sig.Deserialize(&sigBytes); err != nil {
		return false
	}
	for i, key := range blsCommittee.keys {
		if bitmask[i/8]&(byte(1)<<(i%8)) != 0 {
			signerKeys[signerCount] = key
			signerCount++
		}
	}
	return bls.FastAggregateVerify(signerKeys[:signerCount], signingRoot[:], &sig)
}

type dummySyncCommittee [32]byte

// dummyVerifier implements committeeSigVerifier
type dummyVerifier struct{}

// deserializeSyncCommittee implements committeeSigVerifier
func (dummyVerifier) deserializeSyncCommittee(enc []byte) syncCommittee {
	if len(enc) != SerializedCommitteeSize {
		log.Error("Wrong input size for deserializeSyncCommittee", "expected", SerializedCommitteeSize, "got", len(enc))
		return nil
	}
	var sc dummySyncCommittee
	copy(sc[:], enc[:32])
	return sc
}

// verifySignature implements committeeSigVerifier
func (dummyVerifier) verifySignature(committee syncCommittee, signingRoot common.Hash, bitmask, signature []byte) bool {
	return bytes.Equal(signature, makeDummySignature(committee.(dummySyncCommittee), signingRoot, bitmask))
}

func randomDummySyncCommittee() dummySyncCommittee {
	var sc dummySyncCommittee
	rand.Read(sc[:])
	return sc
}

func serializeDummySyncCommittee(sc dummySyncCommittee) []byte {
	enc := make([]byte, SerializedCommitteeSize)
	copy(enc[:32], sc[:])
	return enc
}

func makeDummySignature(committee dummySyncCommittee, signingRoot common.Hash, bitmask []byte) []byte {
	sig := make([]byte, params.BlsSignatureSize)
	for i, b := range committee[:] {
		sig[i] = b ^ signingRoot[i]
	}
	copy(sig[32:], bitmask)
	return sig
}

// Fork describes a single beacon chain fork and also stores the calculated
// signature domain used after this fork.
type Fork struct {
	Epoch uint64 // epoch when given fork version is activated
	Name  string // name of the fork in the chain config (config.yaml) file
	// See fork version definition here:
	//  https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#custom-types
	Version []byte       // fork version
	domain  merkle.Value // calculated by computeDomain, based on fork version and genesis validators root
}

// Forks is the list of all beacon chain forks in the chain configuration.
type Forks []Fork

// domain returns the signature domain for the given epoch (assumes that domains
// have already been calculated).
func (bf Forks) domain(epoch uint64) merkle.Value {
	for i := len(bf) - 1; i >= 0; i-- {
		if epoch >= bf[i].Epoch {
			return bf[i].domain
		}
	}
	log.Error("Fork domain unknown", "epoch", epoch)
	return merkle.Value{}
}

// computeDomain returns the signature domain based on the given fork version
// and genesis validator set root
func computeDomain(forkVersion []byte, genesisValidatorsRoot common.Hash) merkle.Value {
	var (
		hasher        = sha256.New()
		forkVersion32 merkle.Value
		forkDataRoot  merkle.Value
		domain        merkle.Value
	)
	copy(forkVersion32[:len(forkVersion)], forkVersion)
	hasher.Write(forkVersion32[:])
	hasher.Write(genesisValidatorsRoot[:])
	hasher.Sum(forkDataRoot[:0])
	domain[0] = 7
	copy(domain[4:], forkDataRoot[:28])
	return domain
}

// computeDomains calculates and stores signature domains for each fork in the list.
func (bf Forks) computeDomains(genesisValidatorsRoot common.Hash) {
	for i := range bf {
		bf[i].domain = computeDomain(bf[i].Version, genesisValidatorsRoot)
	}
}

// signingRoot calculates the signing root of the given header.
func (bf Forks) signingRoot(header types.Header) common.Hash {
	var (
		signingRoot common.Hash
		headerHash  = header.Hash()
		hasher      = sha256.New()
		domain      = bf.domain(header.Epoch())
	)
	hasher.Write(headerHash[:])
	hasher.Write(domain[:])
	hasher.Sum(signingRoot[:0])
	return signingRoot
}

func (f Forks) Len() int           { return len(f) }
func (f Forks) Swap(i, j int)      { f[i], f[j] = f[j], f[i] }
func (f Forks) Less(i, j int) bool { return f[i].Epoch < f[j].Epoch }

// fieldValue checks if the given fork parameter field is present in the given line
// and if it is then returns the field value and the name of the fork it belongs to.
func fieldValue(line, field string) (name, value string, ok bool) {
	if pos := strings.Index(line, field); pos >= 0 {
		cutFrom := strings.Index(line, "#") // cut in-line comments
		if cutFrom < 0 {
			cutFrom = len(line)
		}
		return line[:pos], strings.TrimSpace(line[pos+len(field) : cutFrom]), true
	}
	return "", "", false
}

// LoadForks parses the beacon chain configuration file (config.yaml) and extracts
// the list of forks
func LoadForks(fileName string) (Forks, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("Error opening beacon chain config file: %v", err)
	}
	defer file.Close()
	var (
		forks        Forks
		forkVersions = make(map[string][]byte)
		forkEpochs   = make(map[string]uint64)
		reader       = bufio.NewReader(file)
	)
	forkEpochs["GENESIS"] = 0

	for {
		l, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Error reading beacon chain config file: %v", err)
		}
		line := string(l)
		if name, value, ok := fieldValue(line, "_FORK_VERSION:"); ok {
			if v, err := hexutil.Decode(value); err == nil {
				forkVersions[name] = v
			} else {
				return nil, fmt.Errorf("Error decoding hex fork id \"%s\" in beacon chain config file: %v", value, err)
			}
		}
		if name, value, ok := fieldValue(line, "_FORK_EPOCH:"); ok {
			if v, err := strconv.ParseUint(value, 10, 64); err == nil {
				forkEpochs[name] = v
			} else {
				return nil, fmt.Errorf("Error parsing epoch number \"%s\" in beacon chain config file: %v", value, err)
			}
		}
	}

	for name, epoch := range forkEpochs {
		if version, ok := forkVersions[name]; ok {
			delete(forkVersions, name)
			forks = append(forks, Fork{Epoch: epoch, Name: name, Version: version})
		} else {
			return nil, fmt.Errorf("Fork id missing for \"%s\" in beacon chain config file", name)
		}
	}

	for name := range forkVersions {
		return nil, fmt.Errorf("Epoch number missing for fork \"%s\" in beacon chain config file", name)
	}
	sort.Sort(forks)
	return forks, nil
}
