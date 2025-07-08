package gmp

// Uses system-installed GMP library

// #cgo LDFLAGS: -lgmp
// #include <gmp.h>
// #include <stdlib.h>
// 
// static inline int mpz_sgn_wrapper(const mpz_t op) {
//     return mpz_sgn(op);
// }
import "C"
import (
    "runtime"
    "unsafe"
)

// Int represents a GMP integer
type Int struct {
    mpz C.mpz_t
}

// NewInt creates a new GMP integer
func NewInt() *Int {
    z := &Int{}
    C.mpz_init(&z.mpz[0])
    runtime.SetFinalizer(z, (*Int).destroy)
    return z
}

// destroy cleans up the GMP integer
func (z *Int) destroy() {
    C.mpz_clear(&z.mpz[0])
}

// SetString sets the integer from a string in the given base
func (z *Int) SetString(s string, base int) (*Int, bool) {
    cs := C.CString(s)
    defer C.free(unsafe.Pointer(cs))
    
    if C.mpz_set_str(&z.mpz[0], cs, C.int(base)) != 0 {
        return nil, false
    }
    return z, true
}

// ExpMod computes z = base^exp mod mod (modular exponentiation)
func (z *Int) ExpMod(base, exp, mod *Int) *Int {
    C.mpz_powm(&z.mpz[0], &base.mpz[0], &exp.mpz[0], &mod.mpz[0])
    return z
}

// SetBytes sets z to the value of buf interpreted as a big-endian unsigned integer
func (z *Int) SetBytes(buf []byte) *Int {
    if len(buf) == 0 {
        C.mpz_set_ui(&z.mpz[0], 0)
        return z
    }
    
    // Use GMP's import function for efficiency
    C.mpz_import(&z.mpz[0], C.size_t(len(buf)), 1, 1, 0, 0, unsafe.Pointer(&buf[0]))
    return z
}

// Bytes returns the absolute value of z as a big-endian byte slice
func (z *Int) Bytes() []byte {
    if z == nil {
        return nil
    }
    
    // Special case: zero returns empty slice (matching big.Int)
    if C.mpz_sgn_wrapper(&z.mpz[0]) == 0 {
        return []byte{}
    }
    
    // Get the number of bytes needed
    size := (C.mpz_sizeinbase(&z.mpz[0], 2) + 7) / 8
    
    // Allocate buffer
    buf := make([]byte, size)
    
    // Export to bytes
    var count C.size_t
    C.mpz_export(unsafe.Pointer(&buf[0]), &count, 1, 1, 0, 0, &z.mpz[0])
    
    // Trim if needed (shouldn't happen but just in case)
    if int(count) < len(buf) {
        buf = buf[:count]
    }
    
    return buf
}

// String returns the decimal representation of z
func (z *Int) String() string {
    if z == nil {
        return "<nil>"
    }
    
    // Get string from GMP
    cs := C.mpz_get_str(nil, 10, &z.mpz[0])
    defer C.free(unsafe.Pointer(cs))
    
    return C.GoString(cs)
}

// BitLen returns the number of bits required to represent z
func (z *Int) BitLen() int {
    if z == nil {
        return 0
    }
    return int(C.mpz_sizeinbase(&z.mpz[0], 2))
}

// Mod sets z to x mod y and returns z
func (z *Int) Mod(x, y *Int) *Int {
    C.mpz_mod(&z.mpz[0], &x.mpz[0], &y.mpz[0])
    return z
}

// SetUint64 sets z to the value of x
func (z *Int) SetUint64(x uint64) *Int {
    C.mpz_set_ui(&z.mpz[0], C.ulong(x))
    return z
}

// ModExp performs modular exponentiation on byte arrays using GMP
// result = base^exp mod mod
// This function matches the behavior of the EVM modexp precompile
func ModExp(base, exp, mod []byte) ([]byte, error) {
    // Handle empty modulus - return empty result (EVM behavior)
    if len(mod) == 0 {
        return []byte{}, nil
    }

    // Check for zero modulus
    allZero := true
    for _, b := range mod {
        if b != 0 {
            allZero = false
            break
        }
    }
    if allZero {
        return []byte{}, nil
    }

    // Create GMP integers
    baseInt := NewInt()
    expInt := NewInt()
    modInt := NewInt()
    resultInt := NewInt()

    // Set values
    baseInt.SetBytes(base)
    expInt.SetBytes(exp)
    modInt.SetBytes(mod)

    // Special case: base has bit length 1 (base == 1)
    if baseInt.BitLen() == 1 {
        // Just return base % mod
        resultInt.Mod(baseInt, modInt)
    } else {
        // Normal case: perform modular exponentiation
        resultInt.ExpMod(baseInt, expInt, modInt)
    }

    // Get result bytes
    return resultInt.Bytes(), nil
}