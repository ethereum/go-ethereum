# EVM Error Handling System

## Overview

The EVM error handling system has been enhanced with structured error types that provide better context and debugging information for EVM execution failures. This document describes the new error types, their usage, and migration guidelines.

## New Error Types

### GasError

Represents gas-related errors with detailed context about gas requirements and availability.

```go
type GasError struct {
    Required  uint64 // Gas required for the operation
    Available uint64 // Gas available for the operation
    Operation string // Name of the operation that failed
}
```

**Usage Example:**
```go
// Creating a gas error
err := NewGasError("SSTORE", 20000, 5000)
// Output: "gas error in SSTORE: required 20000, available 5000"

// Checking for gas errors
if gasErr, ok := err.(*GasError); ok {
    log.Printf("Gas shortage: need %d, have %d", gasErr.Required, gasErr.Available)
}
```

### StackError

Represents stack-related errors with information about stack requirements and current state.

```go
type StackError struct {
    Operation  string    // Name of the operation that failed
    Required   int       // Number of stack items required
    Available  int       // Number of stack items available
    StackTrace []uint64  // Optional stack trace for debugging
}
```

**Usage Example:**
```go
// Creating a stack error
err := NewStackError("ADD", 2, 1, nil)
// Output: "stack error in ADD: required 2 items, available 1"

// With stack trace
stackTrace := []uint64{0x123, 0x456, 0x789}
err := NewStackError("MUL", 2, 0, stackTrace)
```

### MemoryError

Represents memory-related errors with detailed information about memory access patterns.

```go
type MemoryError struct {
    Operation string // Name of the operation that failed
    Requested uint64 // Number of bytes requested
    Available uint64 // Number of bytes available
    Offset    uint64 // Memory offset where the error occurred
}
```

**Usage Example:**
```go
// Creating a memory error
err := NewMemoryError("RETURNDATACOPY", 1024, 512, 0x100)
// Output: "memory error in RETURNDATACOPY: requested 1024 bytes at offset 256, available 512"
```

## Integration with Existing Errors

The new error types work alongside existing EVM errors. The system maintains backward compatibility while providing enhanced error information where applicable.

### Error Wrapping

The new error types can be wrapped with VMError for additional error code information:

```go
gasErr := NewGasError("CALL", 21000, 5000)
vmErr := VMErrorFromErr(gasErr)
```

## Best Practices

### 1. Use Structured Errors for New Code

When implementing new EVM operations or modifying existing ones, prefer structured errors:

```go
// Good: Provides context
if gas < required {
    return NewGasError(opName, required, gas)
}

// Less ideal: Generic error
if gas < required {
    return ErrOutOfGas
}
```

### 2. Include Operation Context

Always include the operation name when creating structured errors:

```go
// Operation name helps with debugging
err := NewStackError("SWAP1", 2, stack.len(), nil)
```

### 3. Add Stack Traces for Complex Operations

For operations that involve multiple steps, consider adding stack traces:

```go
stackTrace := scope.Stack.Data()[:min(10, len(scope.Stack.Data()))]
err := NewStackError("CALL", 7, stack.len(), stackTrace)
```

## Migration Guide

### For Existing Code

1. **Identify Error Creation Points**: Find places where basic errors are created
2. **Add Context**: Replace with structured errors where appropriate
3. **Update Error Handling**: Modify error handling code to work with new types

### Example Migration

**Before:**
```go
if stack.len() < 2 {
    return nil, ErrStackUnderflow
}
```

**After:**
```go
if stack.len() < 2 {
    return nil, NewStackError("ADD", 2, stack.len(), nil)
}
```

### Backward Compatibility

Existing error handling code continues to work. New structured errors implement the `error` interface and can be used anywhere regular errors are expected.

## Error Handling Patterns

### Type Assertion

```go
switch e := err.(type) {
case *GasError:
    // Handle gas-specific error
    log.Printf("Gas error: %s needs %d gas, %d available", 
        e.Operation, e.Required, e.Available)
case *StackError:
    // Handle stack-specific error
    log.Printf("Stack error: %s needs %d items, %d available", 
        e.Operation, e.Required, e.Available)
case *MemoryError:
    // Handle memory-specific error
    log.Printf("Memory error: %s requested %d bytes at offset %d", 
        e.Operation, e.Requested, e.Offset)
default:
    // Handle other errors
    log.Printf("General error: %v", err)
}
```

### Error Checking with errors.As

```go
var gasErr *GasError
if errors.As(err, &gasErr) {
    // Handle gas error specifically
    if gasErr.Required > gasErr.Available*2 {
        // Handle severe gas shortage
    }
}
```

## Testing

### Unit Tests

Test structured errors to ensure proper message formatting and field values:

```go
func TestGasError(t *testing.T) {
    err := NewGasError("TEST", 1000, 500)
    gasErr, ok := err.(*GasError)
    assert.True(t, ok)
    assert.Equal(t, "TEST", gasErr.Operation)
    assert.Equal(t, uint64(1000), gasErr.Required)
    assert.Equal(t, uint64(500), gasErr.Available)
}
```

### Integration Tests

Test error handling in realistic EVM execution scenarios:

```go
func TestStackUnderflowWithContext(t *testing.T) {
    // Setup EVM with insufficient stack
    // Execute operation
    // Verify structured error is returned
}
```

## Performance Considerations

- Error creation is optimized for the failure case
- Structured errors have minimal overhead compared to basic errors
- Stack traces are optional and should be used judiciously
- Error message formatting is lazy (only when .Error() is called)

## Future Enhancements

- Error aggregation for batch operations
- Error recovery mechanisms
- Enhanced debugging information
- Error metrics and monitoring integration 