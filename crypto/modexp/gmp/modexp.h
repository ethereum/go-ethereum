#ifndef MODEXP_H
#define MODEXP_H

#include <stdint.h>
#include <stddef.h>

// Perform modular exponentiation: base^exp mod mod
// Returns 0 on success, -1 on invalid input, -2 if result buffer too small
int modexp_bytes(
    const uint8_t* base, size_t base_len,
    const uint8_t* exp, size_t exp_len,
    const uint8_t* mod, size_t mod_len,
    uint8_t* result, size_t* result_len);

#endif // MODEXP_H