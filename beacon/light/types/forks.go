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

package types

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/beacon/merkle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/go-yaml/yaml"
	"github.com/minio/sha256-simd"
)

const syncCommitteeDomain = 7

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

// Fork returns the fork belonging to the given epoch
func (bf Forks) Fork(epoch uint64) (Fork, bool) {
	for i := len(bf) - 1; i >= 0; i-- {
		if epoch >= bf[i].Epoch {
			return bf[i], true
		}
	}
	return Fork{}, false
}

// domain returns the signature domain for the given epoch (assumes that domains
// have already been calculated).
func (bf Forks) domain(epoch uint64) merkle.Value {
	fork, ok := bf.Fork(epoch)
	if !ok {
		log.Error("Fork domain unknown", "epoch", epoch)
	}
	return fork.domain
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
	domain[0] = syncCommitteeDomain
	copy(domain[4:], forkDataRoot[:28])
	return domain
}

// computeDomains calculates and stores signature domains for each fork in the list.
func (bf Forks) ComputeDomains(genesisValidatorsRoot common.Hash) {
	for i := range bf {
		bf[i].domain = computeDomain(bf[i].Version, genesisValidatorsRoot)
	}
}

// signingRoot calculates the signing root of the given header.
func (bf Forks) SigningRoot(header Header) common.Hash {
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

// LoadForks parses the beacon chain configuration file (config.yaml) and extracts
// the list of forks
func LoadForks(fileName string) (Forks, error) {
	file, err := os.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("Error reading beacon chain config file: %v", err)
	}
	config := make(map[string]string)
	if err := yaml.Unmarshal(file, &config); err != nil {
		return nil, fmt.Errorf("Error parsing beacon chain config YAML file: %v", err)
	}

	var (
		forks        Forks
		forkVersions = make(map[string][]byte)
		forkEpochs   = make(map[string]uint64)
	)
	forkEpochs["GENESIS"] = 0

	for key, value := range config {
		if strings.HasSuffix(key, "_FORK_VERSION") {
			name := key[:len(key)-len("_FORK_VERSION")]
			if v, err := hexutil.Decode(value); err == nil {
				forkVersions[name] = v
			} else {
				return nil, fmt.Errorf("Error decoding hex fork id \"%s\" in beacon chain config file: %v", value, err)
			}
		}
		if strings.HasSuffix(key, "_FORK_EPOCH") {
			name := key[:len(key)-len("_FORK_EPOCH")]
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
