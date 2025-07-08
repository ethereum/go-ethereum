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
//     mpz_init(base_mpz);
//     mpz_init(exp_mpz);
//     mpz_init(mod_mpz);
//     mpz_init(result_mpz);
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
//     // Special case: modulus is zero - return empty result (EVM behavior)
//     if (mpz_cmp_ui(mod_mpz, 0) == 0) {
//         *result_len = 0;
//         mpz_clear(base_mpz);
//         mpz_clear(exp_mpz);
//         mpz_clear(mod_mpz);
//         mpz_clear(result_mpz);
//         return 0;
//     }
//     
//     // Special case: base has bit length 1 (base == 1)
//     // Just return base % mod
//     if (mpz_sizeinbase(base_mpz, 2) == 1) {
//         mpz_mod(result_mpz, base_mpz, mod_mpz);
//     } else {
//         // Normal case: perform modular exponentiation
//         mpz_powm(result_mpz, base_mpz, exp_mpz, mod_mpz);
//     }
//     
//     // Get size needed for result
//     size_t needed = (mpz_sizeinbase(result_mpz, 2) + 7) / 8;
//     if (needed == 0) needed = 1; // For zero result
//     
//     // Check if result buffer is large enough
//     if (*result_len < needed) {
//         mpz_clear(base_mpz);
//         mpz_clear(exp_mpz);
//         mpz_clear(mod_mpz);
//         mpz_clear(result_mpz);
//         return -2;
//     }
//     
//     // Export result to big-endian byte array
//     size_t count;
//     mpz_export(result, &count, 1, 1, 0, 0, result_mpz);
//     *result_len = count;
//     
//     // Handle zero result specially
//     if (count == 0) {
//         result[0] = 0;
//         *result_len = 1;
//     }
//     
//     // Clean up
//     mpz_clear(base_mpz);
//     mpz_clear(exp_mpz);
//     mpz_clear(mod_mpz);
//     mpz_clear(result_mpz);
//     
//     return 0;
// }
import "C"
import (
    "errors"
    "unsafe"
)


// ModExp performs modular exponentiation using the C implementation directly
// This is a lower-level function that bypasses the Go wrapper types
func ModExp(base, exp, mod []byte) ([]byte, error) {
    // Handle empty modulus - return empty result (EVM behavior)
    if len(mod) == 0 {
        return []byte{}, nil
    }
    
    // Allocate result buffer (size of modulus is the max possible result)
    result := make([]byte, len(mod))
    resultLen := C.size_t(len(result))
    
    // Handle empty slices - pass a dummy non-nil pointer with length 0
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