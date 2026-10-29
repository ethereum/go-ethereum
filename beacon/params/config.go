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

package params

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"

	"github.com/ethereum/go-ethereum/beacon/merkle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"gopkg.in/yaml.v3"
)

// syncCommitteeDomain specifies the signatures specific use to avoid clashes
// across signing different data structures.
const syncCommitteeDomain = 7

var knownForks = []string{"GENESIS", "ALTAIR", "BELLATRIX", "CAPELLA", "DENEB", "ELECTRA", "FULU", "GLOAS"}

// ClientConfig contains beacon light client configuration.
type ClientConfig struct {
	ChainConfig
	Apis         []string
	CustomHeader map[string]string
	Threshold    int
	NoFilter     bool
}

// ChainConfig contains the beacon chain configuration.
type ChainConfig struct {
	GenesisTime           uint64      // Unix timestamp of slot 0
	GenesisValidatorsRoot common.Hash // Root hash of the genesis validator set, used for signature domain calculation
	Forks                 Forks
	Checkpoint            common.Hash
	CheckpointFile        string
}

// ForkAtEpoch returns the latest active fork at the given epoch.
func (c *ChainConfig) ForkAtEpoch(epoch uint64) Fork {
	for i := len(c.Forks) - 1; i >= 0; i-- {
		if c.Forks[i].Epoch <= epoch {
			return *c.Forks[i]
		}
	}
	return Fork{}
}

// ForkNameAtSlot returns the name of the latest active fork at the given slot, in
// lower case (as the beacon API names versions).
func (c *ChainConfig) ForkNameAtSlot(slot uint64) string {
	return strings.ToLower(c.ForkAtEpoch(slot / EpochLength).Name)
}

// AddFork adds a new item to the list of forks.
func (c *ChainConfig) AddFork(name string, epoch uint64, version []byte) *ChainConfig {
	knownIndex := slices.Index(knownForks, name)
	if knownIndex == -1 {
		knownIndex = math.MaxInt // assume that the unknown fork happens after the known ones
		if epoch != math.MaxUint64 {
			log.Warn("Unknown fork in config.yaml", "fork name", name, "known forks", knownForks)
		}
	}
	fork := &Fork{
		Name:       name,
		Epoch:      epoch,
		Version:    version,
		knownIndex: knownIndex,
	}
	fork.computeDomain(c.GenesisValidatorsRoot)
	c.Forks = append(c.Forks, fork)
	sort.Sort(c.Forks)
	return c
}

// LoadForks parses the beacon chain configuration file (config.yaml) and extracts
// the list of forks.
func (c *ChainConfig) LoadForks(file []byte) error {
	config := make(map[string]any)
	if err := yaml.Unmarshal(file, &config); err != nil {
		return fmt.Errorf("failed to parse beacon chain config file: %v", err)
	}
	var (
		versions = make(map[string][]byte)
		epochs   = make(map[string]uint64)
	)
	epochs["GENESIS"] = 0

	for key, value := range config {
		if value == nil {
			continue
		}
		if strings.HasSuffix(key, "_FORK_VERSION") {
			name := key[:len(key)-len("_FORK_VERSION")]
			switch version := value.(type) {
			case int:
				versions[name] = new(big.Int).SetUint64(uint64(version)).FillBytes(make([]byte, 4))
			case int64:
				versions[name] = new(big.Int).SetUint64(uint64(version)).FillBytes(make([]byte, 4))
			case uint64:
				versions[name] = new(big.Int).SetUint64(version).FillBytes(make([]byte, 4))
			case string:
				v, err := hexutil.Decode(version)
				if err != nil {
					return fmt.Errorf("failed to decode hex fork id %q in beacon chain config file: %v", version, err)
				}
				versions[name] = v
			default:
				return fmt.Errorf("invalid fork version %q in beacon chain config file", version)
			}
		}
		if strings.HasSuffix(key, "_FORK_EPOCH") {
			name := key[:len(key)-len("_FORK_EPOCH")]
			switch epoch := value.(type) {
			case int:
				epochs[name] = uint64(epoch)
			case int64:
				epochs[name] = uint64(epoch)
			case uint64:
				epochs[name] = epoch
			case string:
				v, err := strconv.ParseUint(epoch, 10, 64)
				if err != nil {
					return fmt.Errorf("failed to parse epoch number %q in beacon chain config file: %v", epoch, err)
				}
				epochs[name] = v
			default:
				return fmt.Errorf("invalid fork epoch %q in beacon chain config file", epoch)
			}
		}
	}
	for name, epoch := range epochs {
		if version, ok := versions[name]; ok {
			delete(versions, name)
			c.AddFork(name, epoch, version)
		} else {
			return fmt.Errorf("fork id missing for %q in beacon chain config file", name)
		}
	}
	for name := range versions {
		return fmt.Errorf("epoch number missing for fork %q in beacon chain config file", name)
	}
	return nil
}

// Fork describes a single beacon chain fork and also stores the calculated
// signature domain used after this fork.
type Fork struct {
	// Name of the fork in the chain config (config.yaml) file
	Name string

	// Epoch when given fork version is activated
	Epoch uint64

	// Fork version, see https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#custom-types
	Version []byte

	// index in list of known forks or MaxInt if unknown
	knownIndex int

	// calculated by computeDomain, based on fork version and genesis validators root
	domain merkle.Value
}

// computeDomain returns the signature domain based on the given fork version
// and genesis validator set root.
func (f *Fork) computeDomain(genesisValidatorsRoot common.Hash) {
	var (
		hasher        = sha256.New()
		forkVersion32 merkle.Value
		forkDataRoot  merkle.Value
	)
	copy(forkVersion32[:], f.Version)
	hasher.Write(forkVersion32[:])
	hasher.Write(genesisValidatorsRoot[:])
	hasher.Sum(forkDataRoot[:0])

	f.domain[0] = syncCommitteeDomain
	copy(f.domain[4:], forkDataRoot[:28])
}

// Forks is the list of all beacon chain forks in the chain configuration.
type Forks []*Fork

// domain returns the signature domain for the given epoch (assumes that domains
// have already been calculated).
func (f Forks) domain(epoch uint64) (merkle.Value, error) {
	for i := len(f) - 1; i >= 0; i-- {
		if epoch >= f[i].Epoch {
			return f[i].domain, nil
		}
	}
	return merkle.Value{}, fmt.Errorf("unknown fork for epoch %d", epoch)
}

// SigningRoot calculates the signing root of the given header.
func (f Forks) SigningRoot(epoch uint64, root common.Hash) (common.Hash, error) {
	domain, err := f.domain(epoch)
	if err != nil {
		return common.Hash{}, err
	}
	var (
		signingRoot common.Hash
		hasher      = sha256.New()
	)
	hasher.Write(root[:])
	hasher.Write(domain[:])
	hasher.Sum(signingRoot[:0])

	return signingRoot, nil
}

func (f Forks) Len() int      { return len(f) }
func (f Forks) Swap(i, j int) { f[i], f[j] = f[j], f[i] }
func (f Forks) Less(i, j int) bool {
	if f[i].Epoch != f[j].Epoch {
		return f[i].Epoch < f[j].Epoch
	}
	return f[i].knownIndex < f[j].knownIndex
}

// SetCheckpointFile sets the checkpoint import/export file name and attempts to
// read the checkpoint from the file if it already exists. It returns true if
// a checkpoint has been loaded. An empty file is ignored, like a missing one.
func (c *ChainConfig) SetCheckpointFile(checkpointFile string) (bool, error) {
	c.CheckpointFile = checkpointFile
	file, err := os.ReadFile(checkpointFile)
	if os.IsNotExist(err) {
		return false, nil // did not load checkpoint
	}
	if err != nil {
		return false, fmt.Errorf("failed to read beacon checkpoint file: %v", err)
	}
	if len(file) == 0 {
		// A crash while the file was written in place can leave it empty, and
		// an empty file may also have been created ahead of the first start.
		log.Warn("Ignoring empty beacon checkpoint file", "file", checkpointFile)
		return false, nil
	}
	cp, err := hexutil.Decode(string(file))
	if err != nil {
		return false, fmt.Errorf("failed to decode hex string in beacon checkpoint file: %v", err)
	}
	if len(cp) != 32 {
		return false, fmt.Errorf("invalid hex string length in beacon checkpoint file: %d", len(cp))
	}
	copy(c.Checkpoint[:len(cp)], cp)
	return true, nil
}

// SaveCheckpointToFile saves the given checkpoint to file if a checkpoint
// import/export file has been specified.
func (c *ChainConfig) SaveCheckpointToFile(checkpoint common.Hash) (bool, error) {
	if c.CheckpointFile == "" {
		return false, nil
	}
	data := []byte(checkpoint.Hex())
	err := replaceFile(c.CheckpointFile, data)
	if errors.Is(err, errNotReplaceable) {
		// A file that can't be replaced at all (see notReplaceable): write over the
		// old checkpoint instead. Warned about once per process.
		if !inPlaceWarned.Swap(true) {
			log.Warn("Saving beacon checkpoint file in place", "file", c.CheckpointFile, "err", err)
		}
		err = overwriteFile(c.CheckpointFile, data)
	}
	if err != nil {
		return false, fmt.Errorf("failed to write beacon checkpoint file: %v", err)
	}
	return true, nil
}

var (
	// errNotReplaceable reports that a file can't be replaced by a new one.
	errNotReplaceable = errors.New("can't replace the file")

	// inPlaceWarned is set once a checkpoint file has been saved in place.
	inPlaceWarned atomic.Bool
)

// replaceFile replaces the named file, or the file a symbolic link leads to (also
// one that doesn't exist yet), with a new one holding data, with the old one's
// permission bits. The new file is written and synced next to the old one first,
// so that a crash leaves either the old or the new content where renaming is
// atomic, as on Linux and macOS. The directory isn't synced: a crash can undo
// the rename, which leaves the old file. Where the file can't be replaced at all,
// it returns errNotReplaceable. On any error the file is as it was, or, on a first
// save, empty (see resolveLink).
func replaceFile(name string, data []byte) error {
	name, err := resolveLink(name)
	if err != nil {
		return err
	}
	perm := os.FileMode(0600)
	if info, err := os.Stat(name); err == nil {
		// Only a file: renaming over a device or a pipe (/dev/null, say) would
		// replace it.
		if m := info.Mode(); m.IsDir() || m&(os.ModeDevice|os.ModeNamedPipe|os.ModeSocket) != 0 {
			return fmt.Errorf("%s is not a regular file", name)
		}
		perm = info.Mode().Perm()
	}
	// A fixed pattern, not the file's name: a long name could make it too long.
	f, err := os.CreateTemp(filepath.Dir(name), ".beacon-checkpoint-*.tmp")
	if err != nil {
		return notReplaceable(err)
	}
	_, err = f.Write(data)
	if err == nil {
		err = f.Chmod(perm)
	}
	if err == nil {
		err = f.Sync()
	}
	if err1 := f.Close(); err1 != nil && err == nil {
		err = err1
	}
	if err == nil {
		if err = os.Rename(f.Name(), name); err != nil {
			err = notReplaceable(err)
		}
	}
	if err != nil {
		os.Remove(f.Name())
	}
	return err
}

// notReplaceable marks the errors that mean the file can't be replaced by a new
// one at all, so that writing over it in place is what's left: no permission (a
// directory that isn't writable, a file held open on Windows), a read-only file
// system (the directory's, the file itself mounted from elsewhere), a busy file
// (a mount point), a file on another file system. Other errors, such as a full
// disk, fail the save and leave the old file as it was.
func notReplaceable(err error) error {
	if errors.Is(err, fs.ErrPermission) || errors.Is(err, syscall.EROFS) ||
		errors.Is(err, syscall.EBUSY) || errors.Is(err, syscall.EXDEV) {
		return fmt.Errorf("%w: %v", errNotReplaceable, err)
	}
	return err
}

// resolveLink returns the file a save replaces: the one name's symbolic links lead
// to, or name itself. A file that doesn't exist yet, directly or behind a link, is
// created first through name, empty (which the loader ignores), so that the system
// resolves the path, links and "..", as it does for every other access.
func resolveLink(name string) (string, error) {
	if runtime.GOOS == "windows" {
		// Windows takes ".." away before it follows links, unlike the walk below.
		name = filepath.Clean(name)
	}
	target, err := filepath.EvalSymlinks(name)
	if err == nil {
		return target, nil
	}
	// (On Windows, a path under a junction comes here too: the walk's ENOTDIR is
	// ERROR_PATH_NOT_FOUND, which also means not-exist.)
	if errors.Is(err, fs.ErrNotExist) {
		// Not O_TRUNC: a file a concurrent save created since stays as it is.
		f, ferr := os.OpenFile(name, os.O_WRONLY|os.O_CREATE, 0600)
		if ferr != nil {
			return "", ferr
		}
		f.Close()
		if target, err = filepath.EvalSymlinks(name); err == nil {
			return target, nil
		}
	}
	// A path EvalSymlinks can't walk (under a Windows junction, say).
	info, lerr := os.Lstat(name)
	switch {
	case errors.Is(lerr, fs.ErrNotExist):
		return name, nil // a new file: create it as named
	case lerr != nil:
		return "", lerr
	case info.Mode()&os.ModeSymlink == 0:
		return name, nil // a file: replace it as named
	case errors.Is(err, syscall.ENOTDIR):
		// A link: write through it, in place.
		return "", fmt.Errorf("%w: %s doesn't resolve: %v", errNotReplaceable, name, err)
	default:
		return "", err
	}
}

// overwriteFile writes data over the beginning of the named file, cuts the file
// to the length of data and syncs it. It doesn't truncate the file first, so a
// file as long as data, as checkpoint files are, never becomes empty.
func overwriteFile(name string, data []byte) error {
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	_, err = f.WriteAt(data, 0)
	if err == nil {
		err = f.Truncate(int64(len(data)))
	}
	if err == nil {
		err = f.Sync()
	}
	if err1 := f.Close(); err1 != nil && err == nil {
		err = err1
	}
	return err
}
