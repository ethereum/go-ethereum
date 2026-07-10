#!/usr/bin/env python3
"""Minimal SlaveConnection protocol peer for Go compatibility tests.

This peer implements only the slave-to-slave protocol that XshardConn needs:
  - Frame read/write (0-byte metadata, matching ReadFrameNoMeta/WriteFrameNoMeta)
  - PING/PONG identity exchange (ClusterOp 0x81/0x82)
  - Echo RPC (opcode → opcode+1, same rpc_id, same payload)

Usage:
    python3 peer.py --port 0 --id "py" --shards "1,2" [--send-ping]

    --port       TCP port to listen on (0 = random, actual port printed to stdout)
    --id         Peer identity (string, encoded as UTF-8 bytes)
    --shards     Comma-separated list of full shard IDs, e.g. "1,2"
    --send-ping  Send PING immediately after connect, wait for PONG, then enter read loop

Output:
    PORT:<port>           Printed when listening
    PONG_OK id=<hex>      Printed when --send-ping PONG is received
    PING_RECEIVED ...     Printed when PING is received from peer
    DISCONNECTED           Printed when connection closes

Behavior:
  - Listens on TCP, accepts one connection
  - If --send-ping: sends PING (rpc_id=1), waits for PONG, prints PONG_OK
  - Read loop:
      PING(0x81) → record peer identity, reply PONG(0x82)
      any opcode → reply opcode+1, same rpc_id, same payload
  - On disconnect: exits
"""
import argparse
import socket
import struct
import sys

from frame import read_frame, write_frame
from messages import (
    serialize_ping_request,
    serialize_pong_response,
    parse_ping_request,
    parse_pong_response,
)

CLUSTER_OP_PING = 0x81
CLUSTER_OP_PONG = 0x82


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--port', type=int, required=True)
    parser.add_argument('--id', type=str, required=True)
    parser.add_argument('--shards', type=str, required=True)
    parser.add_argument('--send-ping', action='store_true')
    args = parser.parse_args()

    peer_id = args.id.encode('utf-8')
    shard_list = [int(s) for s in args.shards.split(',')]

    server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    server.bind(('127.0.0.1', args.port))
    server.listen(1)

    actual_port = server.getsockname()[1]
    print(f"PORT:{actual_port}", flush=True)

    conn, addr = server.accept()

    try:
        if args.send_ping:
            _do_send_ping(conn, peer_id, shard_list)

        _read_loop(conn, peer_id, shard_list)

    except (ConnectionError, BrokenPipeError, OSError):
        pass
    finally:
        conn.close()
        server.close()
        print("DISCONNECTED", flush=True)


def _do_send_ping(conn, peer_id, shard_list):
    """Send PING (rpc_id=1), wait for PONG, validate and print result."""
    ping_payload = serialize_ping_request(peer_id, shard_list)
    write_frame(conn, CLUSTER_OP_PING, 1, ping_payload)

    frame = read_frame(conn)
    if frame is None:
        print("ERROR: no pong received", flush=True)
        sys.exit(1)

    opcode, rpc_id, payload = frame
    if opcode != CLUSTER_OP_PONG:
        print(f"ERROR: expected PONG(0x{CLUSTER_OP_PONG:02x}), got 0x{opcode:02x}", flush=True)
        sys.exit(1)
    if rpc_id != 1:
        print(f"ERROR: expected rpc_id 1, got {rpc_id}", flush=True)
        sys.exit(1)

    peer_id_recv, _ = parse_pong_response(payload)
    print(f"PONG_OK id={peer_id_recv.hex()}", flush=True)


def _read_loop(conn, peer_id, shard_list):
    """Read frames, handle PING or echo RPC, until disconnect."""
    while True:
        frame = read_frame(conn)
        if frame is None:
            break

        opcode, rpc_id, payload = frame

        if opcode == CLUSTER_OP_PING:
            peer_id_recv, peer_shards = parse_ping_request(payload)
            shard_str = ",".join(str(s) for s in peer_shards)
            print(f"PING_RECEIVED id={peer_id_recv.hex()} shards={shard_str}", flush=True)

            pong_payload = serialize_pong_response(peer_id, shard_list)
            write_frame(conn, CLUSTER_OP_PONG, rpc_id, pong_payload)
        else:
            # Echo RPC: opcode+1, same rpc_id, same payload
            write_frame(conn, opcode + 1, rpc_id, payload)


if __name__ == '__main__':
    main()