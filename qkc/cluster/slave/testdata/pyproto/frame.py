"""Frame read/write for slave-to-slave protocol (0-byte metadata).

Wire format: [4B payload_len][1B opcode][8B rpc_id][payload]

This matches Go's qkc/cluster/wire ReadFrameNoMeta/WriteFrameNoMeta.
"""
import struct


def read_frame(conn):
    """Read one frame from conn. Returns (opcode, rpc_id, payload) or None on EOF."""
    header = conn.recv(13)  # 4 (payload_len) + 1 (opcode) + 8 (rpc_id)
    if not header:
        return None
    if len(header) < 13:
        raise ConnectionError("truncated frame header")

    payload_len = struct.unpack('>I', header[0:4])[0]
    opcode = header[4]
    rpc_id = struct.unpack('>Q', header[5:13])[0]

    payload = b''
    while len(payload) < payload_len:
        chunk = conn.recv(payload_len - len(payload))
        if not chunk:
            raise ConnectionError("truncated frame payload")
        payload += chunk

    return (opcode, rpc_id, payload)


def write_frame(conn, opcode, rpc_id, payload):
    """Write one frame to conn."""
    header = struct.pack('>I', len(payload)) + bytes([opcode]) + struct.pack('>Q', rpc_id)
    conn.sendall(header + payload)