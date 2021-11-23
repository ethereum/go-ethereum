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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>

package rawdb

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/rlp"
)

const (
	freezerVersion = 1    // The version tag of freezer table structure
	metaLength     = 1024 // The number of bytes allocated for the freezer table metadata
)

var errIncompatibleVersion = errors.New("incompatible version")

type incompatibleError struct {
	version uint16
	expect  uint16
	err     error
}

func newIncompatibleError(version uint16) *incompatibleError {
	return &incompatibleError{
		version: version,
		expect:  freezerVersion,
		err:     errIncompatibleVersion,
	}
}

// Unwrap returns the internal evm error which allows us for further
// analysis outside.
func (err *incompatibleError) Unwrap() error {
	return err.err
}

func (err *incompatibleError) Error() string {
	return fmt.Sprintf("%v, get %d, expect %d", err.err, err.version, err.expect)
}

// freezerTableMeta wraps all the metadata of the freezer table.
type freezerTableMeta struct {
	version uint16 // Freezer table version descriptor
	tailId  uint32 // The number of the earliest file
	deleted uint64 // The number of items that have been removed from the table
	hidden  uint64 // The number of items that have been hidden in the table
}

// newMetadata initializes the metadata object with the given parameters.
func newMetadata(tailId uint32, deleted uint64, hidden uint64) *freezerTableMeta {
	return &freezerTableMeta{
		version: freezerVersion,
		tailId:  tailId,
		deleted: deleted,
		hidden:  hidden,
	}
}

// encodeMetadata encodes the given parameters as the freezer table metadata.
func encodeMetadata(meta *freezerTableMeta) ([]byte, error) {
	buffer := new(bytes.Buffer)
	if err := rlp.Encode(buffer, meta.version); err != nil {
		return nil, err
	}
	if err := rlp.Encode(buffer, meta.tailId); err != nil {
		return nil, err
	}
	if err := rlp.Encode(buffer, meta.deleted); err != nil {
		return nil, err
	}
	if err := rlp.Encode(buffer, meta.hidden); err != nil {
		return nil, err
	}
	buffer.Write(make([]byte, metaLength-buffer.Len())) // Right pad zero bytes to the specified length
	return buffer.Bytes(), nil
}

// decodeMetadata decodes the freezer-table metadata from the given
// rlp stream.
func decodeMetadata(r *rlp.Stream) (*freezerTableMeta, error) {
	var version uint16
	if err := r.Decode(&version); err != nil {
		return nil, err
	}
	if version != freezerVersion {
		return nil, newIncompatibleError(version)
	}
	var tailId uint32
	if err := r.Decode(&tailId); err != nil {
		return nil, err
	}
	var deleted, hidden uint64
	if err := r.Decode(&deleted); err != nil {
		return nil, err
	}
	if err := r.Decode(&hidden); err != nil {
		return nil, err
	}
	return newMetadata(tailId, deleted, hidden), nil
}

// storeMetadata stores the metadata of the freezer table into the
// given index file.
func storeMetadata(index *os.File, meta *freezerTableMeta) error {
	encoded, err := encodeMetadata(meta)
	if err != nil {
		return err
	}
	if _, err := index.WriteAt(encoded, 0); err != nil {
		return err
	}
	return nil
}

// loadMetadata loads the metadata of the freezer table from the
// given index file. Return the error if the version of loaded
// metadata is not expected.
func loadMetadata(index *os.File) (*freezerTableMeta, error) {
	stat, err := index.Stat()
	if err != nil {
		return nil, err
	}
	if stat.Size() < metaLength {
		return nil, newIncompatibleError(0)
	}
	buffer := make([]byte, metaLength)
	if _, err := index.ReadAt(buffer, 0); err != nil {
		return nil, err
	}
	return decodeMetadata(rlp.NewStream(bytes.NewReader(buffer), 0))
}

// upgradeV0TableIndex extracts the indexes from version-0 index file and
// encodes/stores them into the latest version index file.
func upgradeV0TableIndex(index *os.File) error {
	// Create a temporary offset buffer to read indexEntry info
	buffer := make([]byte, indexEntrySize)

	// Read index zero, determine what file is the earliest
	// and how many entries are deleted from the freezer table.
	var first indexEntry
	if _, err := index.ReadAt(buffer, 0); err != nil {
		return err
	}
	first.unmarshalBinary(buffer)

	encoded, err := encodeMetadata(newMetadata(first.filenum, uint64(first.offset), 0))
	if err != nil {
		return err
	}
	// Close the origin index file.
	if err := index.Close(); err != nil {
		return err
	}
	return copyFrom(index.Name(), index.Name(), indexEntrySize, func(f *os.File) error {
		_, err := f.Write(encoded)
		return err
	})
}

// upgradeTableIndex upgrades the legacy index file to the latest version.
// This function should be responsible for closing the origin index file
// and return the re-opened one.
func upgradeTableIndex(index *os.File, version uint16) (*os.File, *freezerTableMeta, error) {
	switch version {
	case 0:
		if err := upgradeV0TableIndex(index); err != nil {
			return nil, nil, err
		}
	default:
		return nil, nil, errors.New("unknown freezer table index")
	}
	// Reopen the upgraded index file and load the metadata from it
	index, err := os.Open(index.Name())
	if err != nil {
		return nil, nil, err
	}
	meta, err := loadMetadata(index)
	if err != nil {
		return nil, nil, err
	}
	return index, meta, nil
}

// repairTableIndex repairs the given index file of freezer table and returns
// the stored metadata inside. If the index file is be rewritten, the function
// should be responsible for closing the origin one and return the new handler.
// If the table is empty, commit the empty metadata;
// If the table is legacy, upgrade it to the latest version;
func repairTableIndex(index *os.File) (*os.File, *freezerTableMeta, error) {
	stat, err := index.Stat()
	if err != nil {
		return nil, nil, err
	}
	if stat.Size() == 0 {
		meta := newMetadata(0, 0, 0)
		if err := storeMetadata(index, meta); err != nil {
			return nil, nil, err
		}
		// Shift file cursor to the end for next write operation
		_, err = index.Seek(0, 2)
		if err != nil {
			return nil, nil, err
		}
		return index, meta, nil
	}
	meta, err := loadMetadata(index)
	if err != nil {
		if !errors.Is(err, errIncompatibleVersion) {
			return nil, nil, err
		}
		index, meta, err = upgradeTableIndex(index, err.(*incompatibleError).version)
	}
	if err != nil {
		return nil, nil, err
	}
	return index, meta, nil
}
