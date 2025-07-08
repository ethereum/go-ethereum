package gmp

// #include <gmp.h>
import "C"
import (
    "sync"
    generic "github.com/ethereum/go-ethereum/crypto/modexp/gmp/generic"
)

// IntPool provides a pool of reusable GMP Int objects to reduce allocations
type IntPool struct {
    pool sync.Pool
}

// NewIntPool creates a new pool for GMP Int objects
func NewIntPool() *IntPool {
    return &IntPool{
        pool: sync.Pool{
            New: func() interface{} {
                return generic.NewInt()
            },
        },
    }
}

// Get retrieves an Int from the pool
func (p *IntPool) Get() *generic.Int {
    return p.pool.Get().(*generic.Int)
}

// Put returns an Int to the pool after clearing it
func (p *IntPool) Put(i *generic.Int) {
    // Clear the Int to avoid keeping large numbers in memory
    i.SetUint64(0)
    p.pool.Put(i)
}

// ExpModPooled performs modular exponentiation using pooled Int objects
// This is useful for high-throughput scenarios where you want to minimize allocations
// This function matches the behavior of the EVM modexp precompile
func ExpModPooled(pool *IntPool, base, exp, mod []byte) []byte {
    // Handle empty modulus - return empty result (EVM behavior)
    if len(mod) == 0 {
        return []byte{}
    }
    
    // Get Ints from pool
    baseInt := pool.Get()
    expInt := pool.Get()
    modInt := pool.Get()
    resultInt := pool.Get()
    
    // Ensure we return Ints to pool when done
    defer func() {
        pool.Put(baseInt)
        pool.Put(expInt)
        pool.Put(modInt)
        pool.Put(resultInt)
    }()
    
    // Set values
    baseInt.SetBytes(base)
    expInt.SetBytes(exp)
    modInt.SetBytes(mod)
    
    // Check for zero modulus
    if modInt.BitLen() == 0 {
        return []byte{}
    }
    
    // Special case: base has bit length 1 (base == 1)
    if baseInt.BitLen() == 1 {
        // Just return base % mod
        resultInt.Mod(baseInt, modInt)
    } else {
        // Normal case: perform modular exponentiation
        resultInt.ExpMod(baseInt, expInt, modInt)
    }
    
    // Get result bytes (this allocates, but much less than creating new Ints)
    return resultInt.Bytes()
}

// ModExp performs modular exponentiation using a new pool for each operation
// For better performance, create a pool once and reuse it with ExpModPooled
func ModExp(base, exp, mod []byte) ([]byte, error) {
    pool := NewIntPool()
    result := ExpModPooled(pool, base, exp, mod)
    return result, nil
}

// PreallocatedExpMod performs modular exponentiation with pre-allocated Int objects
// This gives the caller full control over object lifecycle
func PreallocatedExpMod(result, base, exp, mod *generic.Int) {
    result.ExpMod(base, exp, mod)
}