/*
  This file is part of ethash.

  ethash is free software: you can redistribute it and/or modify
  it under the terms of the GNU General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  ethash is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU General Public License for more details.

  You should have received a copy of the GNU General Public License
  along with ethash.  If not, see <http://www.gnu.org/licenses/>.
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

#define REVISION 23
#define DATASET_BYTES_INIT 1073741824U // 2**30
#define DATASET_BYTES_GROWTH 8388608U  // 2**23
#define CACHE_BYTES_INIT 1073741824U // 2**24
#define CACHE_BYTES_GROWTH 131072U  // 2**17
#define EPOCH_LENGTH 30000U
#define MIX_BYTES 128
#define HASH_BYTES 64
#define DATASET_PARENTS 256
#define CACHE_ROUNDS 3
#define ACCESSES 64

#ifdef __cplusplus
extern "C" {
#endif

typedef struct ethash_params {
	uint64_t full_size;               // Size of full data set (in bytes, multiple of mix size (128)).
	uint64_t cache_size;              // Size of compute cache (in bytes, multiple of node size (64)).
} ethash_params;

/// Type of a blockhash
typedef struct ethash_blockhash { uint8_t b[32]; } ethash_blockhash_t;
static inline uint8_t ethash_blockhash_get(ethash_blockhash_t const* hash, unsigned int i)
{
    return hash->b[i];
}

static inline void ethash_blockhash_set(ethash_blockhash_t *hash, unsigned int i, uint8_t v)
{
    hash->b[i] = v;
}

static inline void ethash_blockhash_reset(ethash_blockhash_t *hash)
{
    memset(hash, 0, 32);
}

typedef struct ethash_return_value {
    ethash_blockhash_t result;
    ethash_blockhash_t mix_hash;
} ethash_return_value;

uint64_t ethash_get_datasize(const uint32_t block_number);
uint64_t ethash_get_cachesize(const uint32_t block_number);

// initialize the parameters
static inline void ethash_params_init(ethash_params *params, const uint32_t block_number) {
	params->full_size = ethash_get_datasize(block_number);
	params->cache_size = ethash_get_cachesize(block_number);
}

typedef struct ethash_cache {
	void *mem;
} ethash_cache;

void ethash_mkcache(ethash_cache *cache, ethash_params const *params, ethash_blockhash_t const *seed);
void ethash_compute_full_data(void *mem, ethash_params const *params, ethash_cache const *cache);
void ethash_full(ethash_return_value *ret,
                 void const *full_mem,
                 ethash_params const *params,
                 ethash_blockhash_t const *header_hash,
                 const uint64_t nonce);
void ethash_light(ethash_return_value *ret,
                  ethash_cache const *cache,
                  ethash_params const *params,
                  ethash_blockhash_t const *header_hash,
                  const uint64_t nonce);
void ethash_get_seedhash(ethash_blockhash_t *seedhash, const uint32_t block_number);

static inline void ethash_prep_light(void *cache, ethash_params const *params, ethash_blockhash_t const* seed)
{
    ethash_cache c;
    c.mem = cache;
    ethash_mkcache(&c, params, seed);
}

static inline void ethash_compute_light(ethash_return_value *ret, void const *cache, ethash_params const *params, ethash_blockhash_t const *header_hash, const uint64_t nonce)
{
    ethash_cache c;
    c.mem = (void *) cache;
    ethash_light(ret, &c, params, header_hash, nonce);
}

static inline void ethash_prep_full(void *full, ethash_params const *params, void const *cache)
{
    ethash_cache c;
    c.mem = (void *) cache;
    ethash_compute_full_data(full, params, &c);
}

static inline void ethash_compute_full(ethash_return_value *ret,
                                       void const *full,
                                       ethash_params const *params,
                                       ethash_blockhash_t const *header_hash,
                                       const uint64_t nonce)
{
    ethash_full(ret, full, params, header_hash, nonce);
}

// Returns if hash is less than or equal to difficulty
static inline int ethash_check_difficulty(ethash_blockhash_t const *hash,
                                          ethash_blockhash_t const *difficulty)
{
    // Difficulty is big endian
    for (int i = 0; i < 32; i++) {
        if (ethash_blockhash_get(hash, i) == ethash_blockhash_get(difficulty, i)) {
            continue;
        }
        return ethash_blockhash_get(hash, i) < ethash_blockhash_get(difficulty, i);
    }
    return 1;
}

int ethash_quick_check_difficulty(ethash_blockhash_t const *header_hash,
                                  const uint64_t nonce,
                                  ethash_blockhash_t const *mix_hash,
                                  ethash_blockhash_t const *difficulty);

#ifdef __cplusplus
}
#endif
