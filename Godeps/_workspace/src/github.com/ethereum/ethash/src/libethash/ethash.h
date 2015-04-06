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

typedef struct ethash_return_value {
	uint8_t result[32];
	uint8_t mix_hash[32];
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

void ethash_mkcache(ethash_cache *cache, ethash_params const *params, const uint8_t seed[32]);
void ethash_compute_full_data(void *mem, ethash_params const *params, ethash_cache const *cache);
void ethash_full(ethash_return_value *ret, void const *full_mem, ethash_params const *params, const uint8_t header_hash[32], const uint64_t nonce);
void ethash_light(ethash_return_value *ret, ethash_cache const *cache, ethash_params const *params, const uint8_t header_hash[32], const uint64_t nonce);
void ethash_get_seedhash(uint8_t seedhash[32], const uint32_t block_number);

static inline void ethash_prep_light(void *cache, ethash_params const *params, const uint8_t seed[32]) {
	ethash_cache c;
	c.mem = cache;
	ethash_mkcache(&c, params, seed);
}

static inline void ethash_compute_light(ethash_return_value *ret, void const *cache, ethash_params const *params, const uint8_t header_hash[32], const uint64_t nonce) {
	ethash_cache c;
	c.mem = (void *) cache;
	ethash_light(ret, &c, params, header_hash, nonce);
}

static inline void ethash_prep_full(void *full, ethash_params const *params, void const *cache) {
	ethash_cache c;
	c.mem = (void *) cache;
	ethash_compute_full_data(full, params, &c);
}

static inline void ethash_compute_full(ethash_return_value *ret, void const *full, ethash_params const *params, const uint8_t header_hash[32], const uint64_t nonce) {
	ethash_full(ret, full, params, header_hash, nonce);
}

/// @brief Compare two s256-bit big-endian values.
/// @returns 1 if @a a is less than or equal to @a b, 0 otherwise.
/// Both parameters are 256-bit big-endian values.
static inline int ethash_leq_be256(const uint8_t a[32], const uint8_t b[32]) {
	// Boundary is big endian
	for (int i = 0; i < 32; i++) {
		if (a[i] == b[i])
			continue;
		return a[i] < b[i];
	}
	return 1;
}

/// Perofrms a cursory check on the validity of the nonce.
/// @returns 1 if the nonce may possibly be valid for the given header_hash & boundary.
/// @p boundary equivalent to 2 ^ 256 / block_difficulty, represented as a 256-bit big-endian.
int ethash_preliminary_check_boundary(
	const uint8_t header_hash[32],
	const uint64_t nonce,
	const uint8_t mix_hash[32],
	const uint8_t boundary[32]);

#define ethash_quick_check_difficulty ethash_preliminary_check_boundary
#define ethash_check_difficulty ethash_leq_be256

#ifdef __cplusplus
}
#endif
