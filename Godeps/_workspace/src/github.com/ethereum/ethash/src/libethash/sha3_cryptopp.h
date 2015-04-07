#pragma once

#include "compiler.h"
#include <stdint.h>
#include <stdlib.h>

#ifdef __cplusplus
extern "C" {
#endif

struct ethash_blockhash;
typedef struct ethash_blockhash ethash_blockhash_t;

void SHA3_256(ethash_blockhash_t *const ret, const uint8_t *data, size_t size);
void SHA3_512(uint8_t *const ret, const uint8_t *data, size_t size);

#ifdef __cplusplus
}
#endif
