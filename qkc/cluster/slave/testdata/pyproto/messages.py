"""Message serialization matching Go's qkc/serialize + qkc/cluster/wire messages.

Conventions (from Go's serialize package):
  - []byte: 4-byte big-endian length prefix + raw bytes
  - []uint32: 4-byte big-endian count prefix + big-endian uint32 values
  - *RootBlock with ser:"nil": 0x00 = nil
"""
import struct


def serialize_ping_request(id_bytes, full_shard_id_list):
    """Serialize PingRequest matching Go's PingRequest + serialize.

    Fields:
      ID:              []byte       (4B len + raw)
      FullShardIDList: []uint32     (4B count + uint32[])
      RootTip:         *RootBlock   (nil marker 0x00)
    """
    data = b''
    data += struct.pack('>I', len(id_bytes)) + id_bytes
    data += struct.pack('>I', len(full_shard_id_list))
    for shard_id in full_shard_id_list:
        data += struct.pack('>I', shard_id)
    data += b'\x00'  # RootTip: nil
    return data


def serialize_pong_response(id_bytes, full_shard_id_list):
    """Serialize PongResponse matching Go's PongResponse + serialize.

    Fields:
      ID:              []byte       (4B len + raw)
      FullShardIDList: []uint32     (4B count + uint32[])
    """
    data = b''
    data += struct.pack('>I', len(id_bytes)) + id_bytes
    data += struct.pack('>I', len(full_shard_id_list))
    for shard_id in full_shard_id_list:
        data += struct.pack('>I', shard_id)
    return data


def parse_ping_request(data):
    """Parse PingRequest payload. Returns (id, full_shard_id_list)."""
    offset = 0
    id_len = struct.unpack('>I', data[offset:offset + 4])[0]
    offset += 4
    id_bytes = data[offset:offset + id_len]
    offset += id_len
    count = struct.unpack('>I', data[offset:offset + 4])[0]
    offset += 4
    shard_list = []
    for _ in range(count):
        shard_list.append(struct.unpack('>I', data[offset:offset + 4])[0])
        offset += 4
    # Skip RootTip nil marker (1 byte)
    return (id_bytes, shard_list)


def parse_pong_response(data):
    """Parse PongResponse payload. Returns (id, full_shard_id_list)."""
    offset = 0
    id_len = struct.unpack('>I', data[offset:offset + 4])[0]
    offset += 4
    id_bytes = data[offset:offset + id_len]
    offset += id_len
    count = struct.unpack('>I', data[offset:offset + 4])[0]
    offset += 4
    shard_list = []
    for _ in range(count):
        shard_list.append(struct.unpack('>I', data[offset:offset + 4])[0])
        offset += 4
    return (id_bytes, shard_list)