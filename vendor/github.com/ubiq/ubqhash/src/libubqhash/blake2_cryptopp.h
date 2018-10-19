#pragma once

#include "compiler.h"
#include <stdint.h>
#include <stdlib.h>

#ifdef __cplusplus
extern "C" {
#endif

struct ubqhash_h256;

void BLAKE2B_512(uint8_t* const ret, uint8_t const* data, size_t size);

#ifdef __cplusplus
}
#endif
