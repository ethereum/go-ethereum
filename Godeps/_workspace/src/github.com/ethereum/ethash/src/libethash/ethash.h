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

/// Type of a seedhash/blockhash e.t.c.
typedef struct ethash_h256 { uint8_t b[32]; } ethash_h256_t;
static inline uint8_t ethash_h256_get(ethash_h256_t const* hash, unsigned int i)
{
	return hash->b[i];
}

static inline void ethash_h256_set(ethash_h256_t* hash, unsigned int i, uint8_t v)
{
	hash->b[i] = v;
}

static inline void ethash_h256_reset(ethash_h256_t* hash)
{
	memset(hash, 0, 32);
}

// convenience macro to statically initialize an h256_t
// usage:
// ethash_h256_t a = ethash_h256_static_init(1, 2, 3, ... )
// have to provide all 32 values. If you don't provide all the rest
// will simply be unitialized (not guranteed to be 0)
#define ethash_h256_static_init(...)			\
	{ {__VA_ARGS__} }

struct ethash_light;
typedef struct ethash_light* ethash_light_t;
struct ethash_full;
typedef struct ethash_full* ethash_full_t;
typedef int(*ethash_callback_t)(unsigned);

typedef struct ethash_return_value {
	ethash_h256_t result;
	ethash_h256_t mix_hash;
} ethash_return_value_t;

uint64_t ethash_get_datasize(uint64_t const block_number);
uint64_t ethash_get_cachesize(uint64_t const block_number);

typedef struct ethash_cache {
	void* mem;
	uint64_t cache_size;
} ethash_cache_t;

/**
 * Allocate and initialize a new ethash_cache object
 *
 * @param cache_size    The size of the cache in bytes
 * @param seed          Block seedhash to be used during the computation of the
 *                      cache nodes
 * @return              Newly allocated ethash_cache on success or NULL in case of
 *                      ERRNOMEM or invalid parameters used for @ref ethash_compute_cache_nodes()
 */
ethash_cache_t* ethash_cache_new(uint64_t cache_size, ethash_h256_t const* seed);
/**
 * Frees a previously allocated ethash_cache
 * @param c            The object to free
 */
void ethash_cache_delete(ethash_cache_t* c);

/**
 * Allocate and initialize a new ethash_light handler
 *
 * @param cache_size    The size of the cache in bytes
 * @param seed          Block seedhash to be used during the computation of the
 *                      cache nodes
 * @return              Newly allocated ethash_light handler or NULL in case of
 *                      ERRNOMEM or invalid parameters used for @ref ethash_compute_cache_nodes()
 */
ethash_light_t ethash_light_new(uint64_t cache_size, ethash_h256_t const* seed);
/**
 * Frees a previously allocated ethash_light handler
 * @param light        The light handler to free
 */
void ethash_light_delete(ethash_light_t light);
/**
 * Calculate the light client data
 *
 * @param ret            An object of ethash_return_value to hold the return value
 * @param light          The light client handler
 * @param full_size      The size of the full data in bytes.
 * @param header_hash    The header hash to pack into the mix
 * @param nonce          The nonce to pack into the mix
 * @return               true if all went well and false if there were invalid
 *                       parameters given.
 */
bool ethash_light_compute(
	ethash_return_value_t* ret,
	ethash_light_t light,
	uint64_t full_size,
	const ethash_h256_t* header_hash,
	uint64_t const nonce
);
/**
 * Get a pointer to the cache object held by the light client
 *
 * @param light    The light client whose cache to request
 * @return         A pointer to the cache held by the light client or NULL if
 *                 there was no cache in the first place
 */
ethash_cache_t* ethash_light_get_cache(ethash_light_t light);
/**
 * Move the memory ownership of the cache somewhere else
 *
 * @param light    The light client whose cache's memory ownership  to acquire.
 *                 After this function concludes it will no longer have a cache.
 * @return         A pointer to the moved cache or NULL if there was no cache in the first place
 */
ethash_cache_t* ethash_light_acquire_cache(ethash_light_t light);

/**
 * Allocate and initialize a new ethash_full handler
 *
 * @param dirname        The directory in which to put the DAG file.
 * @param seedhash       The seed hash of the block. Used in the DAG file naming.
 * @param full_size      The size of the full data in bytes.
 * @param cache          A cache object to use that was allocated with @ref ethash_cache_new().
 *                       Iff this function succeeds the ethash_full_t will take memory
 *                       memory ownership of the cache and free it at deletion. If
 *                       not then the user still has to handle freeing of the cache himself.
 * @param callback       A callback function with signature of @ref ethash_callback_t
 *                       It accepts an unsigned with which a progress of DAG calculation
 *                       can be displayed. If all goes well the callback should return 0.
 *                       If a non-zero value is returned then DAG generation will stop.
 * @return               Newly allocated ethash_full handler or NULL in case of
 *                       ERRNOMEM or invalid parameters used for @ref ethash_compute_full_data()
 */
ethash_full_t ethash_full_new(
	char const* dirname,
	ethash_h256_t const* seed_hash,
	uint64_t full_size,
	ethash_cache_t const* cache,
	ethash_callback_t callback
);
/**
 * Frees a previously allocated ethash_full handler
 * @param full    The light handler to free
 */
void ethash_full_delete(ethash_full_t full);
/**
 * Calculate the full client data
 *
 * @param ret            An object of ethash_return_value to hold the return value
 * @param full           The full client handler
 * @param header_hash    The header hash to pack into the mix
 * @param nonce          The nonce to pack into the mix
 * @return               true if all went well and false if there were invalid
 *                       parameters given or if there was a callback given and
 *                       at some point return a non-zero value
 */
bool ethash_full_compute(
	ethash_return_value_t* ret,
	ethash_full_t full,
	ethash_h256_t const* header_hash,
	uint64_t const nonce
);
/**
 * Get a pointer to the full DAG data
 */
void *ethash_full_data(ethash_full_t full);

void ethash_get_seedhash(ethash_h256_t *seedhash, const uint32_t block_number);

// Returns if hash is less than or equal to difficulty
static inline int ethash_check_difficulty(
	ethash_h256_t const* hash,
	ethash_h256_t const* difficulty
)
{
	// Difficulty is big endian
	for (int i = 0; i < 32; i++) {
		if (ethash_h256_get(hash, i) == ethash_h256_get(difficulty, i)) {
			continue;
		}
		return ethash_h256_get(hash, i) < ethash_h256_get(difficulty, i);
	}
	return 1;
}

int ethash_quick_check_difficulty(
	ethash_h256_t const* header_hash,
	uint64_t const nonce,
	ethash_h256_t const* mix_hash,
	ethash_h256_t const* difficulty
);

/**
 * Compute the memory data for a full node's memory
 *
 * @param mem         A pointer to an ethash full's memory
 * @param full_size   The size of the full data in bytes
 * @param cache       A cache object to use in the calculation
 * @return            true if all went fine and false for invalid parameters
 */
bool ethash_compute_full_data(void* mem, uint64_t full_size, ethash_cache_t const* cache);


/**
 * =========================
 * =	DEPRECATED API	   =
 * =========================
 *
 * Kept for backwards compatibility with whoever still uses it. Please consider
 * switching to the new API (look above)
 */
typedef struct ethash_params {
 	/// Size of full data set (in bytes, multiple of mix size (128)).
 	uint64_t full_size;
 	/// Size of compute cache (in bytes, multiple of node size (64)).
 	uint64_t cache_size;
} ethash_params;

// initialize the parameters
static inline void ethash_params_init(ethash_params* params, uint32_t const block_number)
{
 	params->full_size = ethash_get_datasize(block_number);
 	params->cache_size = ethash_get_cachesize(block_number);
}

void ethash_mkcache(ethash_cache_t* cache, ethash_params const* params, ethash_h256_t const* seed);
void ethash_full(
	ethash_return_value_t* ret,
	void const* full_mem,
	ethash_params const* params,
	ethash_h256_t const* header_hash,
	uint64_t const nonce
);
void ethash_light(
	ethash_return_value_t* ret,
	ethash_cache_t const* cache,
	ethash_params const* params,
	ethash_h256_t const* header_hash,
	uint64_t const nonce
);

#ifdef __cplusplus
}
#endif
