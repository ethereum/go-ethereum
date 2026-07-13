#!/usr/bin/env python3
"""Minimal MasterConnection protocol peer for Go compatibility tests.

This peer implements the master-to-slave protocol that MasterConn needs:
  - Frame read/write (12-byte ClusterMetadata, matching ReadFrame/WriteFrame)
  - PING/PONG identity exchange (ClusterOp 0x81/0x82)
  - RPC request/response for a representative set of master->slave opcodes
  - Fire-and-forget command dispatch

Usage:
    python3 master.py --port 0 --id "master" --shards "1,2"

Output:
    PORT:<port>                   Printed when listening
    PONG_OK id=<hex>              Printed when PONG is received
    ECO_OK error_code=<n>         Printed when GetEcoInfoListResponse is received
    ROOT_OK error_code=<n>        Printed when AddRootBlockResponse is received
    DESTROY_OK                    Printed after DESTROY command (no response expected)
    DISCONNECTED                  Printed when connection closes

Behavior:
  - Listens on TCP, accepts one connection
  - Sends PING (rpc_id=1), waits for PONG
  - Sends GetEcoInfoListRequest (rpc_id=2), waits for GetEcoInfoListResponse
  - Sends AddRootBlockRequest (rpc_id=3), waits for AddRootBlockResponse
  - Sends DestroyClusterPeerConnectionCommand (rpc_id=0)
  - Sends a second PING (rpc_id=4) to verify the connection is still alive
  - Closes connection and exits
"""
import argparse
import socket
import struct
import sys

from master_frame import read_master_frame, write_master_frame
from messages import serialize_ping_request, parse_pong_response

CLUSTER_OP_BASE = 0x80

CLUSTER_OP_PING = 1 + CLUSTER_OP_BASE
CLUSTER_OP_PONG = 2 + CLUSTER_OP_BASE

# Master -> Slave opcodes
CLUSTER_OP_GET_ECO_INFO_LIST_REQUEST = 7 + CLUSTER_OP_BASE
CLUSTER_OP_GET_ECO_INFO_LIST_RESPONSE = 8 + CLUSTER_OP_BASE
CLUSTER_OP_ADD_ROOT_BLOCK_REQUEST = 5 + CLUSTER_OP_BASE
CLUSTER_OP_ADD_ROOT_BLOCK_RESPONSE = 6 + CLUSTER_OP_BASE
CLUSTER_OP_DESTROY_CLUSTER_PEER_CONNECTION_COMMAND = 27 + CLUSTER_OP_BASE


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--port', type=int, required=True)
    parser.add_argument('--id', type=str, required=True)
    parser.add_argument('--shards', type=str, required=True)
    args = parser.parse_args()

    master_id = args.id.encode('utf-8')
    shard_list = [int(s) for s in args.shards.split(',')]

    server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    server.bind(('127.0.0.1', args.port))
    server.listen(1)

    actual_port = server.getsockname()[1]
    print(f"PORT:{actual_port}", flush=True)

    conn, _ = server.accept()

    try:
        # 1. PING -> PONG
        _send_ping(conn, master_id, shard_list)

        # 2. GetEcoInfoListRequest -> GetEcoInfoListResponse
        _send_rpc(
            conn,
            CLUSTER_OP_GET_ECO_INFO_LIST_REQUEST,
            CLUSTER_OP_GET_ECO_INFO_LIST_RESPONSE,
            2,
            b'',
            'ECO_OK',
        )

        # 3. AddRootBlockRequest -> AddRootBlockResponse
        add_root_block_payload = b'\x00\x00\x00\x00' + b'\x00'  # empty root block + expect_switch=False
        _send_rpc(
            conn,
            CLUSTER_OP_ADD_ROOT_BLOCK_REQUEST,
            CLUSTER_OP_ADD_ROOT_BLOCK_RESPONSE,
            3,
            add_root_block_payload,
            'ROOT_OK',
        )

        # 4. Fire-and-forget DestroyClusterPeerConnectionCommand
        write_master_frame(
            conn,
            CLUSTER_OP_DESTROY_CLUSTER_PEER_CONNECTION_COMMAND,
            0,
            struct.pack('>Q', 42),
            branch=0x00010001,
        )
        print("DESTROY_OK", flush=True)

        # 5. Second PING to confirm the connection is still alive after the
        #    fire-and-forget command.
        _send_ping(conn, master_id, shard_list)

    except (ConnectionError, BrokenPipeError, OSError) as e:
        print(f"ERROR: {e}", flush=True)
        sys.exit(1)
    finally:
        conn.close()
        server.close()
        print("DISCONNECTED", flush=True)


def _send_ping(conn, master_id, shard_list):
    ping_payload = serialize_ping_request(master_id, shard_list)
    write_master_frame(conn, CLUSTER_OP_PING, 1, ping_payload)

    frame = read_master_frame(conn)
    if frame is None:
        print("ERROR: no pong received", flush=True)
        sys.exit(1)

    if frame['opcode'] != CLUSTER_OP_PONG:
        print(f"ERROR: expected PONG(0x{CLUSTER_OP_PONG:02x}), got 0x{frame['opcode']:02x}", flush=True)
        sys.exit(1)
    if frame['rpc_id'] != 1:
        print(f"ERROR: expected rpc_id 1, got {frame['rpc_id']}", flush=True)
        sys.exit(1)

    peer_id_recv, _ = parse_pong_response(frame['payload'])
    print(f"PONG_OK id={peer_id_recv.hex()}", flush=True)


def _send_rpc(conn, req_opcode, resp_opcode, rpc_id, payload, ok_label):
    write_master_frame(conn, req_opcode, rpc_id, payload, branch=0x00010001)

    frame = read_master_frame(conn)
    if frame is None:
        print(f"ERROR: no response for opcode 0x{req_opcode:02x}", flush=True)
        sys.exit(1)

    if frame['opcode'] != resp_opcode:
        print(f"ERROR: expected 0x{resp_opcode:02x}, got 0x{frame['opcode']:02x}", flush=True)
        sys.exit(1)
    if frame['rpc_id'] != rpc_id:
        print(f"ERROR: expected rpc_id {rpc_id}, got {frame['rpc_id']}", flush=True)
        sys.exit(1)

    # First field of these responses is a uint32 error_code.
    if len(frame['payload']) < 4:
        print(f"ERROR: response payload too short for {ok_label}", flush=True)
        sys.exit(1)
    error_code = struct.unpack('>I', frame['payload'][0:4])[0]
    print(f"{ok_label} error_code={error_code}", flush=True)


if __name__ == '__main__':
    main()
