# Adding wire messages

Define message schemas in `messages.go`. Field order is part of the SSZ wire
format. `messages_ssz.go` is generated from those schemas; do not edit it by
hand.

## Workflow

1. Add a unique opcode in `opcode.go` if the operation is new. Use `ClusterOp`
   for cluster RPCs and `CommandOp` for peer commands.
2. Add the message struct to `messages.go` with fields in protocol order. Add
   `ssz-max` tags to variable-length fields and document the inner codec of
   any `Data` field. Use the field rules below when choosing Go types.
3. Register the opcode and message type in `NewClusterMessage` or
   `NewCommandMessage` in `protocol.go`. Register both request and response
   opcodes for an RPC.
4. From the repository root, generate the codecs:

   ```sh
   go generate ./qkc/wire
   ```

5. Add registry, SSZ round-trip, malformed-input, and golden-encoding cases to
   `messages_test.go`, then run `go test ./qkc/wire`.

The `go:generate` directive runs `sszgen` against `messages.go`. It scans the
structs automatically. If a schema uses a named type from another package,
the generator may need that type's source file in its `--include` option;
`common/types.go` is already included for `common.Hash` and `common.Address`.

## Field rules

- **Fixed values:** Use fixed-width integers and arrays, or `ssz-size:"N"` for
  a fixed-size slice. SSZ integers are little-endian; byte arrays are copied
  unchanged.
- **Variable bytes and lists:** `[]byte` and `Data` are byte lists; `[]T` is a
  list of `T`, and `[]Data` is a list of byte lists. Their lengths come from SSZ
  offsets and the enclosing message boundary. Add a numeric `ssz-max` to each
  variable-length field. For nested variable lists, provide one limit per
  dimension and use `ssz-size:"?,?"`.
- **Optional values:** SSZ has no implicit nullable Go pointer. Represent
  presence explicitly in the schema and test both present and absent cases.
- **Limits:** Prefer a protocol-defined `ssz-max` value. The current
  `4294967295` tags are broad generator bounds, not intended message sizes;
  the frame size limit still applies. See the TODO in `messages.go`.

### Encoded objects: `Data`

Use `Data` or `[]Data` when an object cannot be encoded directly by the
generator. Encode the object with its native codec before putting the bytes in
the outer message: RLP for geth objects, SSZ for QKC root objects. For example,
use `rlp.EncodeToBytes(header)` for a geth header or `block.MarshalSSZ()` for
a QKC root block. The outer message's SSZ codec carries `Data` unchanged.

Document the inner codec beside each `Data` field. If construction is easy to
get wrong, add a helper alongside the message. The receiver must decode the
field with the same inner codec.

### Hashes and addresses

Use `common.Hash` for 32-byte hashes and `common.Address` for 20-byte
addresses. They are fixed byte arrays; do not reverse their byte order.

### Numeric uint256

SSZ generation does not support `uint256.Int` directly in these message
schemas. Represent it as `[UInt256Length]byte` and fill the array using
`uint256.Int.MarshalSSZAppend(dst[:0])`. The resulting 32 bytes are
little-endian. Do not apply this numeric byte order rule to hashes.

## Sending a message

After generation, business code passes the wire message and opcode to
`cluster.Sender`. The backend resolves the destination chain ID and uses the
selected connection to send it. `wire.RootChainID` selects the Master from a
chain backend. See [`cluster.Sender`](../cluster/api.go) for the outbound
interface and routing rules. Sending encodes the outer message; `Data` fields
must already contain their correctly encoded inner objects.
