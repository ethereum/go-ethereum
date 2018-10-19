/*
  This file is part of ubqhash.

  ubqhash is free software: you can redistribute it and/or modify
  it under the terms of the GNU General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  ubqhash is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU General Public License for more details.

  You should have received a copy of the GNU General Public License
  along with ubqhash.  If not, see <http://www.gnu.org/licenses/>.
*/

/** @file ubqhash.h
* @date 2015
*/
#pragma once

#include <stdint.h>
#include <stdbool.h>
#include <string.h>
#include <stddef.h>
#include "compiler.h"

#define UBQHASH_REVISION 23
#define UBQHASH_DATASET_BYTES_INIT 1073741824U // 2**30
#define UBQHASH_DATASET_BYTES_GROWTH 8388608U  // 2**23
#define UBQHASH_CACHE_BYTES_INIT 1073741824U // 2**24
#define UBQHASH_CACHE_BYTES_GROWTH 131072U  // 2**17
#define UBQHASH_EPOCH_LENGTH 30000U
#define UBQHASH_MIX_BYTES 128
#define UBQHASH_HASH_BYTES 64
#define UBQHASH_DATASET_PARENTS 256
#define UBQHASH_CACHE_ROUNDS 3
#define UBQHASH_ACCESSES 64
#define UBQHASH_DAG_MAGIC_NUM_SIZE 8
#define UBQHASH_DAG_MAGIC_NUM 0xFEE1DEADBADDCAFE
#define UBQHASH_UIP1_EPOCH 22

#ifdef __cplusplus
extern "C" {
#endif

/// Type of a seedhash/blockhash e.t.c.
typedef struct ubqhash_h256 { uint8_t b[32]; } ubqhash_h256_t;

// convenience macro to statically initialize an h256_t
// usage:
// ubqhash_h256_t a = ubqhash_h256_static_init(1, 2, 3, ... )
// have to provide all 32 values. If you don't provide all the rest
// will simply be unitialized (not guranteed to be 0)
#define ubqhash_h256_static_init(...)			\
	{ {__VA_ARGS__} }

struct ubqhash_light;
typedef struct ubqhash_light* ubqhash_light_t;
struct ubqhash_full;
typedef struct ubqhash_full* ubqhash_full_t;
typedef int(*ubqhash_callback_t)(unsigned);

typedef struct ubqhash_return_value {
	ubqhash_h256_t result;
	ubqhash_h256_t mix_hash;
	bool success;
} ubqhash_return_value_t;

/**
 * Allocate and initialize a new ubqhash_light handler
 *
 * @param block_number   The block number for which to create the handler
 * @return               Newly allocated ubqhash_light handler or NULL in case of
 *                       ERRNOMEM or invalid parameters used for @ref ubqhash_compute_cache_nodes()
 */
ubqhash_light_t ubqhash_light_new(uint64_t block_number);
/**
 * Frees a previously allocated ubqhash_light handler
 * @param light        The light handler to free
 */
void ubqhash_light_delete(ubqhash_light_t light);
/**
 * Calculate the light client data
 *
 * @param light          The light client handler
 * @param header_hash    The header hash to pack into the mix
 * @param nonce          The nonce to pack into the mix
 * @return               an object of ubqhash_return_value_t holding the return values
 */
ubqhash_return_value_t ubqhash_light_compute(
	ubqhash_light_t light,
	ubqhash_h256_t const header_hash,
	uint64_t nonce
);

/**
 * Allocate and initialize a new ubqhash_full handler
 *
 * @param light         The light handler containing the cache.
 * @param callback      A callback function with signature of @ref ubqhash_callback_t
 *                      It accepts an unsigned with which a progress of DAG calculation
 *                      can be displayed. If all goes well the callback should return 0.
 *                      If a non-zero value is returned then DAG generation will stop.
 *                      Be advised. A progress value of 100 means that DAG creation is
 *                      almost complete and that this function will soon return succesfully.
 *                      It does not mean that the function has already had a succesfull return.
 * @return              Newly allocated ubqhash_full handler or NULL in case of
 *                      ERRNOMEM or invalid parameters used for @ref ubqhash_compute_full_data()
 */
ubqhash_full_t ubqhash_full_new(ubqhash_light_t light, ubqhash_callback_t callback);

/**
 * Frees a previously allocated ubqhash_full handler
 * @param full    The light handler to free
 */
void ubqhash_full_delete(ubqhash_full_t full);
/**
 * Calculate the full client data
 *
 * @param full           The full client handler
 * @param header_hash    The header hash to pack into the mix
 * @param nonce          The nonce to pack into the mix
 * @return               An object of ubqhash_return_value to hold the return value
 */
ubqhash_return_value_t ubqhash_full_compute(
	ubqhash_full_t full,
	ubqhash_h256_t const header_hash,
	uint64_t nonce
);
/**
 * Get a pointer to the full DAG data
 */
void const* ubqhash_full_dag(ubqhash_full_t full);
/**
 * Get the size of the DAG data
 */
uint64_t ubqhash_full_dag_size(ubqhash_full_t full);

/**
 * Calculate the seedhash for a given block number
 */
ubqhash_h256_t ubqhash_get_seedhash(uint64_t block_number);

#ifdef __cplusplus
}
#endif
