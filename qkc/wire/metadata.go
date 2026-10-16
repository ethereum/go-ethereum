// Copyright 2026-2027 The quarkchain Authors
// This file is part of the quarkchain library.
//
// The quarkchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The quarkchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the quarkchain library. If not, see <http://www.gnu.org/licenses/>.

package wire

import (
	"encoding/binary"
	"fmt"
)

// ClusterMetadata is the routing header for intra-cluster (master↔slave) traffic.
// Wire representation: 12 bytes (4B chain_id + 8B cluster_peer_id).
//
// Matches pyquarkchain's ClusterMetadata class (protocol.py).
// The base Metadata in pyquarkchain is the 0-byte variant used by direct
// slave-to-slave connections, handled by ReadFrameNoMeta / WriteFrameNoMeta.
type ClusterMetadata struct {
	ChainID       uint32
	ClusterPeerID uint64
}

// MarshalClusterMetadata serializes ClusterMetadata into its 12-byte wire representation.
func MarshalClusterMetadata(m ClusterMetadata) []byte {
	buf := make([]byte, metaSize)
	binary.BigEndian.PutUint32(buf[0:4], m.ChainID)
	binary.BigEndian.PutUint64(buf[4:12], m.ClusterPeerID)
	return buf
}

// UnmarshalClusterMetadata deserializes a 12-byte wire representation into ClusterMetadata.
func UnmarshalClusterMetadata(b []byte) (ClusterMetadata, error) {
	if len(b) != metaSize {
		return ClusterMetadata{}, fmt.Errorf("metadata must be %d bytes, got %d", metaSize, len(b))
	}
	return ClusterMetadata{
		ChainID:       binary.BigEndian.Uint32(b[0:4]),
		ClusterPeerID: binary.BigEndian.Uint64(b[4:12]),
	}, nil
}
