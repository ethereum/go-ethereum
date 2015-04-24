/*
  This file is part of ethash.

  ethash is free software: you can redistribute it and/or modify
  it under the terms of the GNU General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  ethash is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.	See the
  GNU General Public License for more details.

  You should have received a copy of the GNU General Public License
  along with cpp-ethereum.	If not, see <http://www.gnu.org/licenses/>.
*/
/** @file internal.c
* @author Tim Hughes <tim@twistedfury.com>
* @author Matthew Wampler-Doty
* @date 2015
*/

#include <assert.h>
#include <inttypes.h>
#include <stddef.h>
#include <errno.h>
#include <math.h>
#include "mmap.h"
#include "ethash.h"
#include "fnv.h"
#include "endian.h"
#include "internal.h"
#include "data_sizes.h"
#include "io.h"

#ifdef WITH_CRYPTOPP

#include "sha3_cryptopp.h"

#else
#include "sha3.h"
#endif // WITH_CRYPTOPP

uint64_t ethash_get_datasize(uint64_t const block_number)
{
	assert(block_number / EPOCH_LENGTH < 2048);
	return dag_sizes[block_number / EPOCH_LENGTH];
}

uint64_t ethash_get_cachesize(uint64_t const block_number)
{
	assert(block_number / EPOCH_LENGTH < 2048);
	return cache_sizes[block_number / EPOCH_LENGTH];
}

// Follows Sergio's "STRICT MEMORY HARD HASHING FUNCTIONS" (2014)
// https://bitslog.files.wordpress.com/2013/12/memohash-v0-3.pdf
// SeqMemoHash(s, R, N)
bool static ethash_compute_cache_nodes(
	node* const nodes,
	uint64_t cache_size,
	ethash_h256_t const* seed
)
{
	if (cache_size % sizeof(node) != 0) {
		return false;
	}
	uint32_t const num_nodes = (uint32_t) (cache_size / sizeof(node));

	SHA3_512(nodes[0].bytes, (uint8_t*)seed, 32);

	for (unsigned i = 1; i != num_nodes; ++i) {
		SHA3_512(nodes[i].bytes, nodes[i - 1].bytes, 64);
	}

	for (unsigned j = 0; j != CACHE_ROUNDS; j++) {
		for (unsigned i = 0; i != num_nodes; i++) {
			uint32_t const idx = nodes[i].words[0] % num_nodes;
			node data;
			data = nodes[(num_nodes - 1 + i) % num_nodes];
			for (unsigned w = 0; w != NODE_WORDS; ++w) {
				data.words[w] ^= nodes[idx].words[w];
			}
			SHA3_512(nodes[i].bytes, data.bytes, sizeof(data));
		}
	}

	// now perform endian conversion
	fix_endian_arr32(nodes->words, num_nodes * NODE_WORDS);
	return true;
}

ethash_cache_t* ethash_cache_new(uint64_t cache_size, ethash_h256_t const* seed)
{
	ethash_cache_t* ret;
	ret = malloc(sizeof(*ret));
	if (!ret) {
		return NULL;
	}
	ret->mem = malloc((size_t)cache_size);
	if (!ret->mem) {
		goto fail_free_cache;
	}

	node* nodes = (node*)ret->mem;
	if (!ethash_compute_cache_nodes(nodes, cache_size, seed)) {
		goto fail_free_cache_mem;
	}
	ret->cache_size = cache_size;
	return ret;

fail_free_cache_mem:
	free(ret->mem);
fail_free_cache:
	free(ret);
	return NULL;
}

void ethash_cache_delete(ethash_cache_t* c)
{
	free(c->mem);
	free(c);
}

void ethash_calculate_dag_item(
	node* const ret,
	const unsigned node_index,
	ethash_cache_t const* cache
)
{
	uint32_t num_parent_nodes = (uint32_t) (cache->cache_size / sizeof(node));
	node const* cache_nodes = (node const *) cache->mem;
	node const* init = &cache_nodes[node_index % num_parent_nodes];
	memcpy(ret, init, sizeof(node));
	ret->words[0] ^= node_index;
	SHA3_512(ret->bytes, ret->bytes, sizeof(node));
#if defined(_M_X64) && ENABLE_SSE
	__m128i const fnv_prime = _mm_set1_epi32(FNV_PRIME);
	__m128i xmm0 = ret->xmm[0];
	__m128i xmm1 = ret->xmm[1];
	__m128i xmm2 = ret->xmm[2];
	__m128i xmm3 = ret->xmm[3];
#endif

	for (unsigned i = 0; i != DATASET_PARENTS; ++i) {
		uint32_t parent_index = ((node_index ^ i) * FNV_PRIME ^ ret->words[i % NODE_WORDS]) % num_parent_nodes;
		node const *parent = &cache_nodes[parent_index];

#if defined(_M_X64) && ENABLE_SSE
		{
			xmm0 = _mm_mullo_epi32(xmm0, fnv_prime);
			xmm1 = _mm_mullo_epi32(xmm1, fnv_prime);
			xmm2 = _mm_mullo_epi32(xmm2, fnv_prime);
			xmm3 = _mm_mullo_epi32(xmm3, fnv_prime);
			xmm0 = _mm_xor_si128(xmm0, parent->xmm[0]);
			xmm1 = _mm_xor_si128(xmm1, parent->xmm[1]);
			xmm2 = _mm_xor_si128(xmm2, parent->xmm[2]);
			xmm3 = _mm_xor_si128(xmm3, parent->xmm[3]);

			// have to write to ret as values are used to compute index
			ret->xmm[0] = xmm0;
			ret->xmm[1] = xmm1;
			ret->xmm[2] = xmm2;
			ret->xmm[3] = xmm3;
		}
		#else
		{
			for (unsigned w = 0; w != NODE_WORDS; ++w) {
				ret->words[w] = fnv_hash(ret->words[w], parent->words[w]);
			}
		}
#endif
	}
	SHA3_512(ret->bytes, ret->bytes, sizeof(node));
}

bool ethash_compute_full_data(void* mem, uint64_t full_size, ethash_cache_t const* cache)
{
	if (full_size % (sizeof(uint32_t) * MIX_WORDS) != 0 ||
		(full_size % sizeof(node)) != 0) {
		return false;
	}
	node* full_nodes = mem;
	// now compute full nodes
	for (unsigned n = 0; n != (full_size / sizeof(node)); ++n) {
		ethash_calculate_dag_item(&(full_nodes[n]), n, cache);
	}
	return true;
}

static bool ethash_hash(
	ethash_return_value_t* ret,
	node const* full_nodes,
	ethash_cache_t const* cache,
	uint64_t full_size,
	ethash_h256_t const* header_hash,
	uint64_t const nonce,
	ethash_callback_t callback
)
{
	if (full_size % MIX_WORDS != 0) {
		return false;
	}

	// pack hash and nonce together into first 40 bytes of s_mix
	assert(sizeof(node) * 8 == 512);
	node s_mix[MIX_NODES + 1];
	memcpy(s_mix[0].bytes, header_hash, 32);
	fix_endian64(s_mix[0].double_words[4], nonce);

	// compute sha3-512 hash and replicate across mix
	SHA3_512(s_mix->bytes, s_mix->bytes, 40);
	fix_endian_arr32(s_mix[0].words, 16);

	node* const mix = s_mix + 1;
	for (unsigned w = 0; w != MIX_WORDS; ++w) {
		mix->words[w] = s_mix[0].words[w % NODE_WORDS];
	}

	unsigned const page_size = sizeof(uint32_t) * MIX_WORDS;
	unsigned const num_full_pages = (unsigned) (full_size / page_size);

	double const progress_change = 1.0f / ACCESSES / MIX_NODES;
	double progress = 0.0f;
	for (unsigned i = 0; i != ACCESSES; ++i) {
		uint32_t const index = ((s_mix->words[0] ^ i) * FNV_PRIME ^ mix->words[i % MIX_WORDS]) % num_full_pages;

		for (unsigned n = 0; n != MIX_NODES; ++n) {
			node const* dag_node;
			if (callback &&
				callback((unsigned int)(ceil(progress * 100.0f))) != 0) {
				return false;
			}
			progress += progress_change;
			if (full_nodes) {
				dag_node = &full_nodes[MIX_NODES * index + n];
			} else {
				node tmp_node;
				ethash_calculate_dag_item(&tmp_node, index * MIX_NODES + n, cache);
				dag_node = &tmp_node;
			}

#if defined(_M_X64) && ENABLE_SSE
			{
				__m128i fnv_prime = _mm_set1_epi32(FNV_PRIME);
				__m128i xmm0 = _mm_mullo_epi32(fnv_prime, mix[n].xmm[0]);
				__m128i xmm1 = _mm_mullo_epi32(fnv_prime, mix[n].xmm[1]);
				__m128i xmm2 = _mm_mullo_epi32(fnv_prime, mix[n].xmm[2]);
				__m128i xmm3 = _mm_mullo_epi32(fnv_prime, mix[n].xmm[3]);
				mix[n].xmm[0] = _mm_xor_si128(xmm0, dag_node->xmm[0]);
				mix[n].xmm[1] = _mm_xor_si128(xmm1, dag_node->xmm[1]);
				mix[n].xmm[2] = _mm_xor_si128(xmm2, dag_node->xmm[2]);
				mix[n].xmm[3] = _mm_xor_si128(xmm3, dag_node->xmm[3]);
			}
			#else
			{
				for (unsigned w = 0; w != NODE_WORDS; ++w) {
					mix[n].words[w] = fnv_hash(mix[n].words[w], dag_node->words[w]);
				}
			}
#endif
		}

	}

	// compress mix
	for (unsigned w = 0; w != MIX_WORDS; w += 4) {
		uint32_t reduction = mix->words[w + 0];
		reduction = reduction * FNV_PRIME ^ mix->words[w + 1];
		reduction = reduction * FNV_PRIME ^ mix->words[w + 2];
		reduction = reduction * FNV_PRIME ^ mix->words[w + 3];
		mix->words[w / 4] = reduction;
	}

	fix_endian_arr32(mix->words, MIX_WORDS / 4);
	memcpy(&ret->mix_hash, mix->bytes, 32);
	// final Keccak hash
	SHA3_256(&ret->result, s_mix->bytes, 64 + 32); // Keccak-256(s + compressed_mix)
	return true;
}

void ethash_quick_hash(
	ethash_h256_t* return_hash,
	ethash_h256_t const* header_hash,
	uint64_t const nonce,
	ethash_h256_t const* mix_hash
)
{
	uint8_t buf[64 + 32];
	memcpy(buf, header_hash, 32);
	fix_endian64_same(nonce);
	memcpy(&(buf[32]), &nonce, 8);
	SHA3_512(buf, buf, 40);
	memcpy(&(buf[64]), mix_hash, 32);
	SHA3_256(return_hash, buf, 64 + 32);
}

void ethash_get_seedhash(ethash_h256_t* seedhash, const uint32_t block_number)
{
	ethash_h256_reset(seedhash);
	const uint32_t epochs = block_number / EPOCH_LENGTH;
	for (uint32_t i = 0; i < epochs; ++i)
		SHA3_256(seedhash, (uint8_t*)seedhash, 32);
}

int ethash_quick_check_difficulty(
	ethash_h256_t const* header_hash,
	uint64_t const nonce,
	ethash_h256_t const* mix_hash,
	ethash_h256_t const* difficulty
)
{

	ethash_h256_t return_hash;
	ethash_quick_hash(&return_hash, header_hash, nonce, mix_hash);
	return ethash_check_difficulty(&return_hash, difficulty);
}

ethash_light_t ethash_light_new(uint64_t cache_size, ethash_h256_t const* seed)
{
	struct ethash_light *ret;
	ret = calloc(sizeof(*ret), 1);
	if (!ret) {
		return NULL;
	}
	ret->cache = ethash_cache_new(cache_size, seed);
	if (!ret->cache) {
		goto fail_free_light;
	}
	return ret;

fail_free_light:
	free(ret);
	return NULL;
}

void ethash_light_delete(ethash_light_t light)
{
	if (light->cache) {
		ethash_cache_delete(light->cache);
	}
	free(light);
}

bool ethash_light_compute(
	ethash_return_value_t* ret,
	ethash_light_t light,
	uint64_t full_size,
	const ethash_h256_t* header_hash,
	uint64_t const nonce
)
{
	return ethash_hash(
		ret,
		NULL,
		light->cache,
		full_size,
		header_hash,
		nonce,
		NULL
	);
}

ethash_cache_t *ethash_light_get_cache(ethash_light_t light)
{
	return light->cache;
}

ethash_cache_t *ethash_light_acquire_cache(ethash_light_t light)
{
	ethash_cache_t* ret = light->cache;
	light->cache = 0;
	return ret;
}

ethash_full_t ethash_full_new(
	char const* dirname,
	ethash_h256_t const* seed_hash,
	uint64_t full_size,
	ethash_cache_t const* cache,
	ethash_callback_t callback
)
{
	struct ethash_full* ret;
	int fd;
	FILE *f = NULL;
	bool match = false;
	ret = calloc(sizeof(*ret), 1);
	if (!ret) {
		return NULL;
	}
	ret->file_size = (size_t)full_size;
	switch (ethash_io_prepare(dirname, *seed_hash, &f, (size_t)full_size)) {
	case ETHASH_IO_FAIL:
	case ETHASH_IO_MEMO_SIZE_MISMATCH:
		goto fail_free_full;
	case ETHASH_IO_MEMO_MATCH:
		match = true;
	case ETHASH_IO_MEMO_MISMATCH:
		ret->file = f;
		if ((fd = ethash_fileno(ret->file)) == -1) {
			goto fail_free_full;
		}
		ret->data = mmap(
			NULL,
			(size_t)full_size,
			PROT_READ | PROT_WRITE,
			MAP_SHARED,
			fd,
			0
		);
		if (ret->data == MAP_FAILED) {
			goto fail_close_file;
		}
		if (match) {
			return ret;
		}
		break;
	}

	if (!ethash_compute_full_data(ret->data, full_size, cache)) {
		goto fail_free_full_data;
	}
	ret->callback = callback;
	return ret;

fail_free_full_data:
	// could check that munmap(..) == 0 but even if it did not can't really do anything here
	munmap(ret->data, (size_t)full_size);
fail_close_file:
	fclose(ret->file);
fail_free_full:
	free(ret);
	return NULL;
}

void ethash_full_delete(ethash_full_t full)
{
	// could check that munmap(..) == 0 but even if it did not can't really do anything here
	munmap(full->data, full->file_size);
	if (full->file) {
		fclose(full->file);
	}
	free(full);
}

bool ethash_full_compute(
	ethash_return_value_t* ret,
	ethash_full_t full,
	ethash_h256_t const* header_hash,
	uint64_t const nonce
)
{
	return ethash_hash(
		ret,
		(node const*)full->data,
		NULL,
		full->file_size,
		header_hash,
		nonce,
		full->callback
	);
}

void *ethash_full_data(ethash_full_t full)
{
    return full->data;
}

/**
 * =========================
 * =	DEPRECATED API	   =
 * =========================
 *
 * Kept for backwards compatibility with whoever still uses it. Please consider
 * switching to the new API (look above)
 */
void ethash_mkcache(
	ethash_cache_t* cache,
	ethash_params const* params,
	ethash_h256_t const* seed
)
{
	node* nodes = (node*) cache->mem;
	ethash_compute_cache_nodes(nodes, params->cache_size, seed);
}

void ethash_full(
	ethash_return_value_t* ret,
	void const* full_mem,
	ethash_params const* params,
	ethash_h256_t const* header_hash,
	uint64_t const nonce
)
{
	ethash_hash(ret, (node const *) full_mem, NULL, params->full_size, header_hash, nonce, NULL);
}
void ethash_light(
	ethash_return_value_t* ret,
	ethash_cache_t const* cache,
	ethash_params const* params,
	ethash_h256_t const* header_hash,
	uint64_t const nonce
)
{
	ethash_hash(ret, NULL, cache, params->full_size, header_hash, nonce, NULL);
}
