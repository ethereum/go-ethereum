#!/usr/bin/env python3
"""Export EIP-8297 test vectors from the EELS reference implementation.

Regenerates the committed JSON vector files consumed by the Go test suites
in trie/bintrie. Run against a pinned execution-specs checkout (branch
eip-8297-tests):

    PYTHONPATH=$EELS/src $EELS/.venv/bin/python export_vectors.py \
        --eels $EELS --out .

The vectors are committed; this script is the regeneration procedure for
when the EIP (notably its still-open hash choice) or the reference moves.
"""

import argparse
import json
import random
import subprocess
import sys
from pathlib import Path


def eels_commit(eels: Path) -> str:
    out = subprocess.run(
        ["git", "-C", str(eels), "rev-parse", "--short=9", "HEAD"],
        capture_output=True, text=True, check=True)
    return out.stdout.strip()


def hx(b: bytes) -> str:
    return "0x" + b.hexdigest() if hasattr(b, "hexdigest") else "0x" + bytes(b).hex()


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--eels", required=True, type=Path)
    ap.add_argument("--out", default=".", type=Path)
    args = ap.parse_args()

    sys.path.insert(0, str(args.eels / "src"))
    sys.path.insert(0, str(args.eels / "tests" / "binary_trie"))

    from ethereum.binary_trie.trie import (  # noqa: E402
        BinaryTrie, EMPTY_TRIE_ROOT, blake3_hash, root, trie_set)
    from ethereum.binary_trie import embedding  # noqa: E402
    from ethereum_types.bytes import Bytes, Bytes20, Bytes32  # noqa: E402
    from ethereum_types.numeric import U8, U32, U64, U256, Uint  # noqa: E402

    meta = {
        "source": f"execution-specs@{eels_commit(args.eels)} (branch eip-8297-tests)",
        "hasher": "blake3",
        "generator": "export_vectors.py",
    }

    def account_key(i: int, sub: int) -> bytes:
        addr = embedding.address20_to_address32(Bytes20(i.to_bytes(20, "big")))
        return bytes(embedding.get_tree_key_for_header(addr, Uint(sub)))

    def storage_key(i: int, slot: int) -> bytes:
        addr = embedding.address20_to_address32(Bytes20(i.to_bytes(20, "big")))
        return bytes(embedding.get_tree_key_for_storage_slot(addr, U256(slot)))

    def value(i: int) -> bytes:
        return i.to_bytes(32, "big")

    # --- trie_vectors: fixed populations -> root -------------------------
    trie_vectors = []

    def add_trie_vector(name, entries):
        t = BinaryTrie()
        for k, v in entries:
            trie_set(t, Bytes(k), Bytes32(v))
        trie_vectors.append({
            "name": name,
            "entries": [{"key": hx(k), "value": hx(v)} for k, v in entries],
            "root": hx(root(t)),
        })

    add_trie_vector("empty", [])
    add_trie_vector("single_account_leaf", [(account_key(1, 0), value(7))])
    add_trie_vector("one_header_stem_two_leaves", [
        (account_key(1, 0), value(7)), (account_key(1, 1), value(9))])
    add_trie_vector("two_accounts", [
        (account_key(1, 0), value(7)), (account_key(2, 0), value(8))])
    add_trie_vector("cross_zone_small", [
        (account_key(1, 0), value(1)), (account_key(1, 1), value(2)),
        (storage_key(1, 100), value(3)), (storage_key(1, 356), value(4)),
        (account_key(2, 0), value(5))])
    add_trie_vector("zero_value_present", [(account_key(3, 0), b"\x00" * 32)])
    full_stem = [(account_key(4, s), value(s + 1)) for s in range(256)]
    add_trie_vector("full_header_stem", full_stem)

    # --- sequence_vectors: randomized op streams with deletes ------------
    sequence_vectors = []
    for seed in (8297, 11832, 3102, 90210, 20260727):
        rng = random.Random(seed)
        t = BinaryTrie()
        ops, roots = [], []
        live = []
        for step in range(120):
            do_delete = live and rng.random() < 0.25
            if do_delete:
                k = live.pop(rng.randrange(len(live)))
                del t._data[Bytes(k)]
                ops.append({"op": "delete", "key": hx(k)})
            else:
                acct = rng.randrange(1, 40)
                if rng.random() < 0.5:
                    k = account_key(acct, rng.randrange(256))
                else:
                    k = storage_key(acct, rng.randrange(0, 1200))
                v = value(rng.randrange(1, 2**32))
                if k not in [x for x in live]:
                    live.append(k)
                trie_set(t, Bytes(k), Bytes32(v))
                ops.append({"op": "set", "key": hx(k), "value": hx(v)})
            roots.append(hx(root(t)))
        sequence_vectors.append({"seed": seed, "ops": ops, "roots_after": roots})

    # --- embedding_vectors ----------------------------------------------
    addr20 = Bytes20(bytes.fromhex("aa" * 20))
    addr32 = embedding.address20_to_address32(addr20)
    code_hash = Bytes32(bytes.fromhex("bb" * 32))
    embedding_vectors = {
        "address": hx(addr20),
        "basic_data_key": hx(embedding.get_tree_key_for_basic_data(addr32)),
        "code_hash_key": hx(embedding.get_tree_key_for_code_hash(addr32)),
        "slots": [
            {"slot": s, "key": hx(embedding.get_tree_key_for_storage_slot(addr32, U256(s)))}
            for s in (0, 5, 63, 64, 255, 256, 1000, 2**255)
        ],
        "chunks": [
            {"chunk": c, "key": hx(embedding.get_tree_key_for_code_chunk(addr32, code_hash, Uint(c)))}
            for c in (0, 5, 127, 128, 300, 383, 384)
        ],
    }

    # --- basic_data & chunkify vectors -----------------------------------
    basic_data_vectors = [
        {"code_size": cs, "nonce": n, "balance": str(b),
         "value": hx(embedding.encode_basic_data(U32(cs), U64(n), U256(b)))}
        for cs, n, b in [
            (0, 0, 0), (0, 1, 10**18), (287454020, 0x5566778899AABBCC,
                                        0x0123456789ABCDEF0123456789ABCDEF),
            (24576, 1, 1)]
    ]
    chunk_cases = {
        "empty": b"",
        "short": bytes.fromhex("6001"),
        "push_boundary": bytes.fromhex("60" * 62),  # PUSH1 chain across chunk edge
        "push32_tail": bytes([0x7F] + [0xEE] * 40),
        "zeros62": b"\x00" * 62,
    }
    chunkify_vectors = [
        {"name": name, "code": hx(code),
         "chunks": [hx(c) for c in embedding.chunkify_code(Bytes(code))]}
        for name, code in chunk_cases.items()
    ]

    out = {
        "meta": meta,
        "empty_root": hx(EMPTY_TRIE_ROOT),
        "trie_vectors": trie_vectors,
        "sequence_vectors": sequence_vectors,
        "embedding_vectors": embedding_vectors,
        "basic_data_vectors": basic_data_vectors,
        "chunkify_vectors": chunkify_vectors,
    }
    dest = args.out / "eip8297_vectors.json"
    dest.write_text(json.dumps(out, indent=1))
    print(f"wrote {dest} ({dest.stat().st_size} bytes) from {meta['source']}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
