/*
  This file is part of cpp-ethereum.

  cpp-ethereum is free software: you can redistribute it and/or modify
  it under the terms of the GNU General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  cpp-ethereum is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU General Public License for more details.

  You should have received a copy of the GNU General Public License
  along with cpp-ethereum.  If not, see <http://www.gnu.org/licenses/>.
*/
/** @file ethash.h
* @date 2015
*/
#pragma once

#include <stdint.h>
#include <stdbool.h>
#include <string.h>
#include <stddef.h>
#include "compiler.h"

#define REVISION 18
#define DAGSIZE_BYTES_INIT 1073741824U
#define DAG_GROWTH 113000000U
#define EPOCH_LENGTH 30000U
#define MIX_BYTES 128
#define DAG_PARENTS 256
#define CACHE_ROUNDS 3
#define ACCESSES 64

#ifdef __cplusplus
extern "C" {
#endif

typedef struct ethash_params {
    size_t full_size;               // Size of full data set (in bytes, multiple of mix size (128)).
    size_t cache_size;              // Size of compute cache (in bytes, multiple of node size (64)).
} ethash_params;

typedef struct ethash_return_value {
    uint8_t result[32];
    uint8_t mix_hash[32];
} ethash_return_value;

size_t const ethash_get_datasize(const uint32_t block_number);
size_t const ethash_get_cachesize(const uint32_t block_number);

// initialize the parameters
static inline void ethash_params_init(ethash_params *params, const uint32_t block_number) {
    params->full_size = ethash_get_datasize(block_number);
    params->cache_size = ethash_get_cachesize(block_number);
}

typedef struct ethash_cache {
    void *mem;
} ethash_cache;

void ethash_mkcache(ethash_cache *cache, ethash_params const *params, const uint8_t seed[32]);
void ethash_compute_full_data(void *mem, ethash_params const *params, ethash_cache const *cache);
void ethash_full(ethash_return_value *ret, void const *full_mem, ethash_params const *params, const uint8_t header_hash[32], const uint64_t nonce);
void ethash_light(ethash_return_value *ret, ethash_cache const *cache, ethash_params const *params, const uint8_t header_hash[32], const uint64_t nonce);

static inline int ethash_check_difficulty(
        const uint8_t hash[32],
        const uint8_t difficulty[32]) {
    // Difficulty is big endian
    for (int i = 0; i < 32; i++) {
        if (hash[i] == difficulty[i]) continue;
        return hash[i] < difficulty[i];
    }
    return 0;
}

int ethash_quick_check_difficulty(
        const uint8_t header_hash[32],
        const uint64_t nonce,
        const uint8_t mix_hash[32],
        const uint8_t difficulty[32]);

#ifdef __cplusplus
}
#endif
