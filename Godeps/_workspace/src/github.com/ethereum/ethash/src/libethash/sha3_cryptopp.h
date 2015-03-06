#pragma once

#include "compiler.h"
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

void SHA3_256(uint8_t *const ret, const uint8_t *data, size_t size);
void SHA3_512(uint8_t *const ret, const uint8_t *data, size_t size);

#ifdef __cplusplus
}
#endif