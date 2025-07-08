package gmp

import (
    "bytes"
    "math/big"
    "testing"
)

// TestWrapperVsExisting compares the wrapper with existing GMP bindings
func TestWrapperVsExisting(t *testing.T) {
    tests := []struct {
        name string
        base string
        exp  string
        mod  string
    }{
        {
            name: "small_numbers",
            base: "2",
            exp:  "10",
            mod:  "1000",
        },
        {
            name: "medium_numbers",
            base: "123456789",
            exp:  "987654321",
            mod:  "1000000007",
        },
        {
            name: "large_numbers",
            base: "123456789012345678901234567890",
            exp:  "987654321098765432109876543210",
            mod:  "111111111111111111111111111111",
        },
        {
            name: "zero_exponent",
            base: "12345",
            exp:  "0",
            mod:  "67890",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Parse numbers
            baseBig := new(big.Int)
            baseBig.SetString(tt.base, 10)
            expBig := new(big.Int)
            expBig.SetString(tt.exp, 10)
            modBig := new(big.Int)
            modBig.SetString(tt.mod, 10)
            
            // Test with wrapper
            baseBytes := baseBig.Bytes()
            expBytes := expBig.Bytes()
            modBytes := modBig.Bytes()
            
            wrapperResult, err := ModExp(baseBytes, expBytes, modBytes)
            if err != nil {
                t.Fatalf("Wrapper error: %v", err)
            }
            
            // Test with existing GMP bindings
            gmpBase := NewInt()
            gmpBase.SetString(tt.base, 10)
            gmpExp := NewInt()
            gmpExp.SetString(tt.exp, 10)
            gmpMod := NewInt()
            gmpMod.SetString(tt.mod, 10)
            gmpResult := NewInt()
            gmpResult.ExpMod(gmpBase, gmpExp, gmpMod)
            
            existingResult := gmpResult.Bytes()
            
            // Compare results
            if !bytes.Equal(wrapperResult, existingResult) {
                t.Errorf("Results differ:\nWrapper:  %x\nExisting: %x", 
                    wrapperResult, existingResult)
            }
        })
    }
}

// BenchmarkWrapperVsExisting compares performance
func BenchmarkModExp(b *testing.B) {
    base := make([]byte, 60)
    exp := make([]byte, 60)
    mod := make([]byte, 60)
    
    for i := range base {
        base[i] = byte(i * 17)
        exp[i] = byte(i * 31)
        mod[i] = byte(255 - i)
    }
    mod[59] |= 0x01
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = ModExp(base, exp, mod)
    }
}

func BenchmarkModExpExisting(b *testing.B) {
    baseBytes := make([]byte, 60)
    expBytes := make([]byte, 60)
    modBytes := make([]byte, 60)
    
    for i := range baseBytes {
        baseBytes[i] = byte(i * 17)
        expBytes[i] = byte(i * 31)
        modBytes[i] = byte(255 - i)
    }
    modBytes[59] |= 0x01
    
    base := NewInt()
    base.SetBytes(baseBytes)
    exp := NewInt()
    exp.SetBytes(expBytes)
    mod := NewInt()
    mod.SetBytes(modBytes)
    result := NewInt()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        result.ExpMod(base, exp, mod)
        _ = result.Bytes()
    }
}