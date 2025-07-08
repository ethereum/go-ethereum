package gmp

// #cgo LDFLAGS: -lgmp
// #include <gmp.h>
// #include <stdlib.h>
// #include <string.h>
// #include <stdint.h>
//
// static int modexp_bytes(
//     const uint8_t* base, size_t base_len,
//     const uint8_t* exp, size_t exp_len,
//     const uint8_t* mod, size_t mod_len,
//     uint8_t* result, size_t* result_len)
// {
//     mpz_t base_mpz, exp_mpz, mod_mpz, result_mpz;
//     
//     // Check for NULL pointers
//     if (!base || !exp || !mod || !result || !result_len) {
//         return -1;
//     }
//     
//     // Initialize GMP integers
//     mpz_inits(base_mpz, exp_mpz, mod_mpz, result_mpz, NULL);
//     
//     // Import big-endian byte arrays into GMP integers
//     // Handle empty arrays specially - GMP treats NULL with size 0 as 0
//     if (base_len > 0) {
//         mpz_import(base_mpz, base_len, 1, 1, 0, 0, base);
//     }
//     if (exp_len > 0) {
//         mpz_import(exp_mpz, exp_len, 1, 1, 0, 0, exp);
//     }
//     if (mod_len > 0) {
//         mpz_import(mod_mpz, mod_len, 1, 1, 0, 0, mod);
//     }
//     
//     // Perform modular exponentiation
//     mpz_powm(result_mpz, base_mpz, exp_mpz, mod_mpz);
//     
//     // Get exact size needed for result
//     size_t needed = 0;
//     mpz_export(NULL, &needed, 1, 1, 0, 0, result_mpz);
//     
//     // Check if result buffer is large enough
//     if (*result_len < needed) {
//         mpz_clears(base_mpz, exp_mpz, mod_mpz, result_mpz, NULL);
//         return -2;
//     }
//     
//     // Export result to big-endian byte array
//     mpz_export(result, &needed, 1, 1, 0, 0, result_mpz);
//     *result_len = needed;
//     
//     // Clean up
//     mpz_clears(base_mpz, exp_mpz, mod_mpz, result_mpz, NULL);
//     
//     return 0;
// }
import "C"
import (
    "errors"
    "runtime"
    "unsafe"
)


// ModExp performs modular exponentiation using the C implementation directly
// This is a lower-level function that bypasses the Go wrapper types
//
// This is thread safe.
func ModExp(base, exp, mod []byte) ([]byte, error) {
    // Handle empty modulus - return empty result (EVM behavior)
    if len(mod) == 0 {
        return []byte{}, nil
    }
    
    // Special case: zero modulus
    // TODO: Check to see if theres a cleaner way to do this
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
    
    // // Special case: base == 1
    // // Check if base is 1 (only one byte with value 1, or leading zeros followed by 1)
    // baseIsOne := false
    // if len(base) == 0 {
    //     baseIsOne = false
    // } else if len(base) == 1 && base[0] == 1 {
    //     baseIsOne = true
    // } else {
    //     // Check for leading zeros followed by 1
    //     allZeroExceptLast := true
    //     for i := 0; i < len(base)-1; i++ {
    //         if base[i] != 0 {
    //             allZeroExceptLast = false
    //             break
    //         }
    //     }
    //     if allZeroExceptLast && base[len(base)-1] == 1 {
    //         baseIsOne = true
    //     }
    // }
    
    // if baseIsOne {
    //     // base^exp mod mod = 1 mod mod = 1 (if mod > 1), 0 (if mod == 1)
    //     // Just return base % mod which is 1 % mod
    //     if len(mod) == 1 && mod[0] == 1 {
    //         return []byte{}, nil  // 1 % 1 = 0
    //     }
    //     return []byte{1}, nil  // 1 % mod = 1 for mod > 1
    // }
    
    // Allocate result buffer (size of modulus is the max possible result)
    result := make([]byte, len(mod))
    resultLen := C.size_t(len(result))
    
    // Handle empty slices - pass a dummy non-nil pointer with length 0
    // This avoids UB when the length is zero.
    dummy := C.uint8_t(0)
    var basePtr, expPtr, modPtr *C.uint8_t = &dummy, &dummy, &dummy
    
    if len(base) > 0 {
        basePtr = (*C.uint8_t)(unsafe.Pointer(&base[0]))
    }
    if len(exp) > 0 {
        expPtr = (*C.uint8_t)(unsafe.Pointer(&exp[0]))
    }
    if len(mod) > 0 {
        modPtr = (*C.uint8_t)(unsafe.Pointer(&mod[0]))
    }
    
    // Call C function
    ret := C.modexp_bytes(
        basePtr, C.size_t(len(base)),
        expPtr, C.size_t(len(exp)),
        modPtr, C.size_t(len(mod)),
        (*C.uint8_t)(unsafe.Pointer(&result[0])), &resultLen,
    )
    
    // Keep the slices alive until after the C call completes
    runtime.KeepAlive(base)
    runtime.KeepAlive(exp)
    runtime.KeepAlive(mod)
    runtime.KeepAlive(result)
    
    // Check for errors
    switch ret {
    case 0:
        // Success - trim result to actual size
        if resultLen == 0 {
            return []byte{}, nil
        }
        return result[:resultLen], nil
    case -1:
        return nil, errors.New("invalid parameter")
    case -2:
        return nil, errors.New("result buffer too small")
    default:
        return nil, errors.New("unknown error")
    }
}