package gmp

import (
    "math/big"
    "sync"
    "testing"
)

// TestIntPool tests basic pool functionality
func TestIntPool(t *testing.T) {
    pool := NewIntPool()
    
    // Get an Int from pool
    i1 := pool.Get()
    i1.SetString("12345", 10)
    
    // Return it to pool
    pool.Put(i1)
    
    // Get another Int (should be the same one, cleared)
    i2 := pool.Get()
    if i2.String() != "0" {
        t.Errorf("Expected cleared Int, got %s", i2.String())
    }
    
    // Verify it's the same object
    i2.SetString("67890", 10)
    pool.Put(i2)
    
    i3 := pool.Get()
    // Should be cleared again
    if i3.String() != "0" {
        t.Errorf("Expected cleared Int after second put, got %s", i3.String())
    }
}

// TestExpModPooled tests pooled modular exponentiation
func TestExpModPooled(t *testing.T) {
    pool := NewIntPool()
    
    base := []byte{0x02}      // 2
    exp := []byte{0x0A}       // 10
    mod := []byte{0x03, 0xE8} // 1000
    
    result := ExpModPooled(pool, base, exp, mod)
    
    // 2^10 mod 1000 = 1024 mod 1000 = 24
    expected := []byte{0x18} // 24
    if !bytesEqual(result, expected) {
        t.Errorf("ExpModPooled: expected %x, got %x", expected, result)
    }
}

// TestModExp tests the convenience function
func TestModExp(t *testing.T) {
    base := []byte{0x02}
    exp := []byte{0x0A}
    mod := []byte{0x03, 0xE8}
    
    result, err := ModExp(base, exp, mod)
    if err != nil {
        t.Fatalf("ModExp failed: %v", err)
    }
    
    expected := []byte{0x18}
    if !bytesEqual(result, expected) {
        t.Errorf("ModExp: expected %x, got %x", expected, result)
    }
    
    // Test empty modulus case
    result, err = ModExp(base, exp, []byte{})
    if err != nil {
        t.Errorf("Expected no error for empty modulus, got %v", err)
    }
    if len(result) != 0 {
        t.Errorf("Expected empty result for empty modulus, got %x", result)
    }
}


// TestPreallocatedExpMod tests pre-allocated operations
func TestPreallocatedExpMod(t *testing.T) {
    base := NewInt()
    exp := NewInt()
    mod := NewInt()
    result := NewInt()
    
    base.SetString("2", 10)
    exp.SetString("10", 10)
    mod.SetString("1000", 10)
    
    PreallocatedExpMod(result, base, exp, mod)
    
    if result.String() != "24" {
        t.Errorf("PreallocatedExpMod: expected 24, got %s", result.String())
    }
}

// BenchmarkExpModPooled compares pooled vs non-pooled performance
func BenchmarkExpModPooled(b *testing.B) {
    pool := NewIntPool()
    base := []byte("123456789012345678901234567890")
    exp := []byte("987654321098765432109876543210")
    mod := []byte("111111111111111111111111111111")
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = ExpModPooled(pool, base, exp, mod)
    }
}

// BenchmarkExpModNonPooled benchmarks non-pooled operations for comparison
func BenchmarkExpModNonPooled(b *testing.B) {
    base := []byte("123456789012345678901234567890")
    exp := []byte("987654321098765432109876543210")
    mod := []byte("111111111111111111111111111111")
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        baseInt := NewInt()
        expInt := NewInt()
        modInt := NewInt()
        resultInt := NewInt()
        
        baseInt.SetBytes(base)
        expInt.SetBytes(exp)
        modInt.SetBytes(mod)
        resultInt.ExpMod(baseInt, expInt, modInt)
        _ = resultInt.Bytes()
    }
}


// TestPoolConcurrency tests that the pool is safe for concurrent use
func TestPoolConcurrency(t *testing.T) {
    pool := NewIntPool()
    var wg sync.WaitGroup
    
    // Run many concurrent operations
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(n int) {
            defer wg.Done()
            
            // Get and use an Int
            num := pool.Get()
            num.SetString("12345", 10)
            
            // Simulate some work
            result := NewInt()
            result.ExpMod(num, num, num)
            
            // Return to pool
            pool.Put(num)
        }(i)
    }
    
    wg.Wait()
}

// TestPoolWithBigNumbers tests pool with large numbers
func TestPoolWithBigNumbers(t *testing.T) {
    pool := NewIntPool()
    
    // Test with 2048-bit numbers
    base := make([]byte, 256)
    exp := make([]byte, 256)
    mod := make([]byte, 256)
    
    // Fill with test data
    for i := range base {
        base[i] = byte(i)
        exp[i] = byte(255 - i)
        mod[i] = 0xFF
    }
    mod[0] = 0x7F // Make sure modulus is not too large
    
    // This should not panic or leak memory
    for i := 0; i < 10; i++ {
        result := ExpModPooled(pool, base, exp, mod)
        if len(result) == 0 {
            t.Error("Expected non-empty result")
        }
    }
}

// TestPoolComparison verifies pooled operations match non-pooled
func TestPoolComparison(t *testing.T) {
    pool := NewIntPool()
    
    testCases := []struct {
        base string
        exp  string
        mod  string
    }{
        {"2", "10", "1000"},
        {"123456789", "987654321", "1000000007"},
        {"999999999999", "888888888888", "777777777777"},
    }
    
    for _, tc := range testCases {
        // Non-pooled
        base1 := NewInt()
        exp1 := NewInt()
        mod1 := NewInt()
        result1 := NewInt()
        
        base1.SetString(tc.base, 10)
        exp1.SetString(tc.exp, 10)
        mod1.SetString(tc.mod, 10)
        result1.ExpMod(base1, exp1, mod1)
        
        // Pooled
        result2 := ExpModPooled(pool, base1.Bytes(), exp1.Bytes(), mod1.Bytes())
        
        // Compare
        if !bytesEqual(result1.Bytes(), result2) {
            t.Errorf("Pooled vs non-pooled mismatch for %s^%s mod %s", tc.base, tc.exp, tc.mod)
        }
        
        // Also compare with math/big
        bigBase := new(big.Int)
        bigExp := new(big.Int)
        bigMod := new(big.Int)
        bigResult := new(big.Int)
        
        bigBase.SetString(tc.base, 10)
        bigExp.SetString(tc.exp, 10)
        bigMod.SetString(tc.mod, 10)
        bigResult.Exp(bigBase, bigExp, bigMod)
        
        if !bytesEqual(bigResult.Bytes(), result2) {
            t.Errorf("Pooled vs math/big mismatch for %s^%s mod %s", tc.base, tc.exp, tc.mod)
        }
    }
}