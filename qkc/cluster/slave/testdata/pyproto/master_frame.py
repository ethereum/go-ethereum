"""Frame read/write for master-slave protocol (12-byte ClusterMetadata).

Wire format: [4B payload_len][4B branch][8B cluster_peer_id][1B opcode][8B rpc_id][payload]

This matches Go's qkc/cluster/wire ReadFrame/WriteFrame with ClusterMetadata.
"""
import struct


def read_master_frame(conn):
    """Read one master frame from conn.

    Returns a dict with keys: branch, cluster_peer_id, opcode, rpc_id, payload.
    Returns None on EOF.
    """
    header = conn.recv(25)  # 4 + 12 + 1 + 8
    if not header:
        return None
    if len(header) < 25:
        raise ConnectionError("truncated master frame header")

    payload_len = struct.unpack('>I', header[0:4])[0]
    branch = struct.unpack('>I', header[4:8])[0]
    cluster_peer_id = struct.unpack('>Q', header[8:16])[0]
    opcode = header[16]
    rpc_id = struct.unpack('>Q', header[17:25])[0]

    payload = b''
    while len(payload) < payload_len:
        chunk = conn.recv(payload_len - len(payload))
        if not chunk:
            raise ConnectionError("truncated master frame payload")
        payload += chunk

    return {
        'branch': branch,
        'cluster_peer_id': cluster_peer_id,
        'opcode': opcode,
        'rpc_id': rpc_id,
        'payload': payload,
    }


def write_master_frame(conn, opcode, rpc_id, payload, branch=0, cluster_peer_id=0):
    """Write one master frame to conn."""
    header = (
        struct.pack('>I', len(payload))
        + struct.pack('>I', branch)
        + struct.pack('>Q', cluster_peer_id)
        + bytes([opcode])
        + struct.pack('>Q', rpc_id)
    )
    conn.sendall(header + payload)
