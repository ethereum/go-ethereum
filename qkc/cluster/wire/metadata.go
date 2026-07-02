// Copyright 2026-2027, QuarkChain.

package wire

import (
	"encoding/binary"
	"fmt"
)

// ClusterMetadata is the routing header for intra-cluster (master↔slave) traffic.
// Wire representation: 12 bytes (4B branch + 8B cluster_peer_id).
//
// Matches pyquarkchain's ClusterMetadata class (protocol.py).
// The base Metadata in pyquarkchain is the 0-byte variant used by direct
// slave-to-slave connections, handled by ReadFrameNoMeta / WriteFrameNoMeta.
type ClusterMetadata struct {
	Branch        uint32
	ClusterPeerID uint64
}

// MarshalClusterMetadata serializes ClusterMetadata into its 12-byte wire representation.
func MarshalClusterMetadata(m ClusterMetadata) []byte {
	buf := make([]byte, metaSize)
	binary.BigEndian.PutUint32(buf[0:4], m.Branch)
	binary.BigEndian.PutUint64(buf[4:12], m.ClusterPeerID)
	return buf
}

// UnmarshalClusterMetadata deserializes a 12-byte wire representation into ClusterMetadata.
func UnmarshalClusterMetadata(b []byte) (ClusterMetadata, error) {
	if len(b) != metaSize {
		return ClusterMetadata{}, fmt.Errorf("metadata must be %d bytes, got %d", metaSize, len(b))
	}
	return ClusterMetadata{
		Branch:        binary.BigEndian.Uint32(b[0:4]),
		ClusterPeerID: binary.BigEndian.Uint64(b[4:12]),
	}, nil
}
