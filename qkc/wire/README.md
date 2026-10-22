# SSZ message workflow

`messages.go` contains the wire message schemas. Field order is part of each
SSZ container. `messages_ssz.go` is generated code and must not be edited by
hand.

## Add a message

1. Add a Go struct to `messages.go` with fields in protocol order. Use fixed
   width integers and arrays for fixed-size values, `[]byte` for variable byte
   strings, and `[]T` for variable lists. For nested variable lists, use one
   limit per dimension and `ssz-size:"?,?"`.
2. For variable-length fields, add `ssz-max` with a numeric limit. Use a
   protocol-defined limit when one is known, or the current broad limit
   `4294967295` otherwise. Use fixed Go types or `ssz-size:"N"` for
   fixed-size fields.
3. Add the opcode-to-type case to `NewClusterMessage` or
   `NewCommandMessage` in `protocol.go`.
4. From the repository root, run:

   ```sh
   go generate ./qkc/wire
   ```

5. Add codec round-trip, malformed-input, and golden-encoding cases to
   `messages_test.go`.

The `go:generate` directive changes to the repository root before invoking
`sszgen` and passes repository-relative source and output paths. No list of
message type names needs to be maintained in the generator command: it scans
the message source file and generates codecs for its structs.

## SSZ field rules

- `uintN` and `[N]byte` encode as fixed-size values.
- `[]byte` is a `ByteList`; its actual length is determined by SSZ offsets and
  the enclosing message boundary, not by a legacy length prefix.
- `[]T` is a `List[T]`; `[][]byte` is a list of byte lists. Each dynamic list
  uses offsets to locate its elements.
- SSZ integers and offsets are little-endian. The transport layer carries the
  encoded SSZ bytes as the message payload.
- SSZ has no implicit nullable Go pointer. Represent presence explicitly in
  the schema and test both present and absent cases.
For nested variable lists, provide one `ssz-max` value per dimension and use
`ssz-size:"?,?"`. The current `4294967295` tags are broad placeholders and can
be revisited later; see the TODO in `messages.go`.
