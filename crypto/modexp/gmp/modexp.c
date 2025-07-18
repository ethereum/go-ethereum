#include <gmp.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>

int modexp_bytes(
    const uint8_t* base, size_t base_len,
    const uint8_t* exp, size_t exp_len,
    const uint8_t* mod, size_t mod_len,
    uint8_t* result, size_t* result_len)
{
    mpz_t base_mpz, exp_mpz, mod_mpz, result_mpz;
    
    // Check for NULL pointers
    if (!base || !exp || !mod || !result || !result_len) {
        return -1;
    }
    
    // Initialize GMP integers
    mpz_inits(base_mpz, exp_mpz, mod_mpz, result_mpz, NULL);
    
    // Import big-endian byte arrays into GMP integers
    // Handle empty arrays specially - GMP treats NULL with size 0 as 0
    if (base_len > 0) {
        mpz_import(base_mpz, base_len, 1, 1, 0, 0, base);
    }
    if (exp_len > 0) {
        mpz_import(exp_mpz, exp_len, 1, 1, 0, 0, exp);
    }
    if (mod_len > 0) {
        mpz_import(mod_mpz, mod_len, 1, 1, 0, 0, mod);
    }
    
    // Perform modular exponentiation
    mpz_powm(result_mpz, base_mpz, exp_mpz, mod_mpz);
    
    // Get exact size needed for result
    size_t needed = 0;
    mpz_export(NULL, &needed, 1, 1, 0, 0, result_mpz);
    
    // Check if result buffer is large enough
    if (*result_len < needed) {
        mpz_clears(base_mpz, exp_mpz, mod_mpz, result_mpz, NULL);
        return -2;
    }
    
    // Export result to big-endian byte array
    mpz_export(result, &needed, 1, 1, 0, 0, result_mpz);
    *result_len = needed;
    
    // Clean up
    mpz_clears(base_mpz, exp_mpz, mod_mpz, result_mpz, NULL);
    
    return 0;
}