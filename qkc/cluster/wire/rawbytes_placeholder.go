// Copyright 2026-2027, QuarkChain.

// =============================================================================
// WIRE MIGRATION SHIM (NOT PART OF PROTOCOL SPEC)
// =============================================================================
//
// This file provides temporary placeholder types used during the migration
// from Python QuarkChain Serializable types to native Go structs.
//
// This is an IMPLEMENTATION-ONLY migration aid.
// It is NOT part of the wire protocol specification.
//
// -----------------------------------------------------------------------------
// IMPORTANT DISTINCTION
// -----------------------------------------------------------------------------
//
// The wire protocol is defined by the concrete message structs in package wire.
//
// RawBytes does NOT implement the original serialization logic of the Python
// Serializable type it replaces. It only exists to allow incremental migration
// by temporarily representing unported complex types.
//
// -----------------------------------------------------------------------------
// Migration Strategy
// -----------------------------------------------------------------------------
//
// Some Python Serializable types (for example:
//
//   - RootBlock
//   - MinorBlockHeader
//   - TypedTransaction
//   - CrossShardTransactionList
//   - TokenBalanceMap
//
// ) may not yet have corresponding Go implementations.
//
// During migration, these types may temporarily be represented as:
//
//     *RawBytes
//
// This allows:
//   - message structs to be migrated incrementally
//   - Go code to compile before all dependent types are ported
//   - each complex type to be replaced independently
//
// RawBytes is expected to be removed once the corresponding native Go type
// has been implemented.
//
// -----------------------------------------------------------------------------
// RawBytes Semantics
// -----------------------------------------------------------------------------
//
// RawBytes is an opaque placeholder containing serialized bytes of an
// unported Python Serializable object.
//
// It does NOT:
//   - decode the contained data
//   - inspect the contained data
//   - reproduce the original Python serialization format
//   - define any protocol-level wire encoding rules
//
// Serialization behavior is inherited from the generic serialization framework
// for byte slices. The resulting wire format may differ from the original
// Python Serializable encoding.
//
// Any required wire compatibility must be achieved by replacing RawBytes with
// the correct concrete Go type.
//
// -----------------------------------------------------------------------------
// SAFETY CONSTRAINTS
// -----------------------------------------------------------------------------
//
// RawBytes MUST:
//
//   1. remain opaque and must not be partially decoded
//   2. not be used as a permanent protocol type
//   3. be replaced by the corresponding concrete Go implementation
//
// Using RawBytes as a final protocol representation may result in wire format
// incompatibility.
//
// -----------------------------------------------------------------------------
// Lifecycle
// -----------------------------------------------------------------------------
//
// Migration completion:
//
//   1. Implement the corresponding native Go struct
//   2. Replace all *RawBytes fields with concrete types
//   3. Verify compatibility through Python/Go wire compatibility tests
//   4. Remove this migration shim
//
// -----------------------------------------------------------------------------
// WARNING
// -----------------------------------------------------------------------------
//
// This file contains temporary migration helpers only.
// It must not become part of the production protocol implementation.
//

package wire

// RawBytes is an opaque placeholder for Python Serializable types that have
// not yet been migrated to native Go structs.
//
// RawBytes does not define custom wire behavior. It follows the default
// serialization behavior of the underlying byte slice type.
//
// It must be replaced by the corresponding concrete Go type once migration
// is complete.
type RawBytes []byte
