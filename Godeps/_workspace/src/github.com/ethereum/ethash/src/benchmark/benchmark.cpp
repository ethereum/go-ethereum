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
/** @file benchmark.cpp
 * @author Tim Hughes <tim@twistedfury.com>
 * @date 2015
 */

#include <stdio.h>
#include <stdlib.h>
#include <chrono>
#include <libethash/ethash.h>
#include <libethash/util.h>
#ifdef OPENCL
#include <libethash-cl/ethash_cl_miner.h>
#endif
#include <vector>
#include <algorithm>

#ifdef WITH_CRYPTOPP
#include <libethash/sha3_cryptopp.h>
#include <string>

#else
#include "libethash/sha3.h"
#endif // WITH_CRYPTOPP

#undef min
#undef max

using std::chrono::high_resolution_clock;

#if defined(OPENCL)
const unsigned trials = 1024*1024*32;
#elif defined(FULL)
const unsigned trials = 1024*1024/8;
#else
const unsigned trials = 1024*1024/1024;
#endif
uint8_t g_hashes[1024*32];

static char nibbleToChar(unsigned nibble)
{
	return (char) ((nibble >= 10 ? 'a'-10 : '0') + nibble);
}

static uint8_t charToNibble(char chr)
{
	if (chr >= '0' && chr <= '9')
	{
		return (uint8_t) (chr - '0');
	}
	if (chr >= 'a' && chr <= 'z')
	{
		return (uint8_t) (chr - 'a' + 10);
	}
	if (chr >= 'A' && chr <= 'Z')
	{
		return (uint8_t) (chr - 'A' + 10);
	}
	return 0;
}

static std::vector<uint8_t> hexStringToBytes(char const* str)
{
	std::vector<uint8_t> bytes(strlen(str) >> 1);
	for (unsigned i = 0; i != bytes.size(); ++i)
	{
		bytes[i] = charToNibble(str[i*2 | 0]) << 4;
		bytes[i] |= charToNibble(str[i*2 | 1]);
	}
	return bytes;
}

static std::string bytesToHexString(uint8_t const* bytes, unsigned size)
{
	std::string str;
	for (unsigned i = 0; i != size; ++i)
	{
		str += nibbleToChar(bytes[i] >> 4);
		str += nibbleToChar(bytes[i] & 0xf);
	}
	return str;
}

static std::string bytesToHexString(ethash_h256_t const *hash, unsigned size)
{
	return bytesToHexString((uint8_t*)hash, size);
}

extern "C" int main(void)
{
	// params for ethash
	ethash_params params;
	ethash_params_init(&params, 0);
	//params.full_size = 262147 * 4096;	// 1GBish;
	//params.full_size = 32771 * 4096;	// 128MBish;
	//params.full_size = 8209 * 4096;	// 8MBish;
	//params.cache_size = 8209*4096;
	//params.cache_size = 2053*4096;
	ethash_h256_t seed;
	ethash_h256_t previous_hash;

	memcpy(&seed, hexStringToBytes("9410b944535a83d9adf6bbdcc80e051f30676173c16ca0d32d6f1263fc246466").data(), 32);
	memcpy(&previous_hash, hexStringToBytes("c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470").data(), 32);

	// allocate page aligned buffer for dataset
#ifdef FULL
	void* full_mem_buf = malloc(params.full_size + 4095);
	void* full_mem = (void*)((uintptr_t(full_mem_buf) + 4095) & ~4095);
#endif
	void* cache_mem_buf = malloc(params.cache_size + 63);
	void* cache_mem = (void*)((uintptr_t(cache_mem_buf) + 63) & ~63);

	ethash_cache cache;
	cache.mem = cache_mem;

	// compute cache or full data
	{
		auto startTime = high_resolution_clock::now();
		ethash_mkcache(&cache, &params, &seed);
		auto time = std::chrono::duration_cast<std::chrono::milliseconds>(high_resolution_clock::now() - startTime).count();

		ethash_h256_t cache_hash;
		SHA3_256(&cache_hash, (uint8_t const*)cache_mem, params.cache_size);
		debugf("ethash_mkcache: %ums, sha3: %s\n", (unsigned)((time*1000)/CLOCKS_PER_SEC), bytesToHexString(&cache_hash, sizeof(cache_hash)).data());

		// print a couple of test hashes
		{
			auto startTime = high_resolution_clock::now();
			ethash_return_value hash;
			ethash_light(&hash, &cache, &params, &previous_hash, 0);
			auto time = std::chrono::duration_cast<std::chrono::milliseconds>(high_resolution_clock::now() - startTime).count();
			debugf("ethash_light test: %ums, %s\n", (unsigned)time, bytesToHexString(&hash.result, 32).data());
		}

		#ifdef FULL
			startTime = high_resolution_clock::now();
			ethash_compute_full_data(full_mem, &params, &cache);
			time = std::chrono::duration_cast<std::chrono::milliseconds>(high_resolution_clock::now() - startTime).count();
			debugf("ethash_compute_full_data: %ums\n", (unsigned)time);
		#endif // FULL
	}

#ifdef OPENCL
	ethash_cl_miner miner;
	{
		auto startTime = high_resolution_clock::now();
		if (!miner.init(params, &seed))
			exit(-1);
		auto time = std::chrono::duration_cast<std::chrono::milliseconds>(high_resolution_clock::now() - startTime).count();
		debugf("ethash_cl_miner init: %ums\n", (unsigned)time);
	}
#endif


#ifdef FULL
	{
		auto startTime = high_resolution_clock::now();
		ethash_return_value hash;
		ethash_full(&hash, full_mem, &params, &previous_hash, 0);
		auto time = std::chrono::duration_cast<std::chrono::milliseconds>(high_resolution_clock::now() - startTime).count();
		debugf("ethash_full test: %uns\n", (unsigned)time);
	}
#endif

#ifdef OPENCL
	// validate 1024 hashes against CPU
	miner.hash(g_hashes, (uint8_t*)&previous_hash, 0, 1024);
	for (unsigned i = 0; i != 1024; ++i)
	{
		ethash_return_value hash;
		ethash_light(&hash, &cache, &params, &previous_hash, i);
		if (memcmp(&hash.result, g_hashes + 32*i, 32) != 0)
		{
			debugf("nonce %u failed: %s %s\n", i, bytesToHexString(g_hashes + 32*i, 32).c_str(), bytesToHexString(&hash.result, 32).c_str());
			static unsigned c = 0;
			if (++c == 16)
			{
				exit(-1);
			}
		}
	}

	// ensure nothing else is going on
	miner.finish();
#endif

	auto startTime = high_resolution_clock::now();
	unsigned hash_count = trials;

	#ifdef OPENCL
	{
		struct search_hook : ethash_cl_miner::search_hook
		{
			unsigned hash_count;
			std::vector<uint64_t> nonce_vec;

			virtual bool found(uint64_t const* nonces, uint32_t count)
			{
				nonce_vec.insert(nonce_vec.end(), nonces, nonces + count);
				return false;
			}

			virtual bool searched(uint64_t start_nonce, uint32_t count)
			{
				// do nothing
				hash_count += count;
				return hash_count >= trials;
			}
		};
		search_hook hook;
		hook.hash_count = 0;

		miner.search((uint8_t*)&previous_hash, 0x000000ffffffffff, hook);

		for (unsigned i = 0; i != hook.nonce_vec.size(); ++i)
		{
			uint64_t nonce = hook.nonce_vec[i];
			ethash_return_value hash;
			ethash_light(&hash, &cache, &params, &previous_hash, nonce);
			debugf("found: %.8x%.8x -> %s\n", unsigned(nonce>>32), unsigned(nonce), bytesToHexString(&hash.result, 32).c_str());
		}

		hash_count = hook.hash_count;
	}
	#else
	{
		//#pragma omp parallel for
		for (int nonce = 0; nonce < trials; ++nonce)
		{
			ethash_return_value hash;
			#ifdef FULL
				ethash_full(&hash, full_mem, &params, &previous_hash, nonce);
			#else
				ethash_light(&hash, &cache, &params, &previous_hash, nonce);
			#endif // FULL
		}
	}
	#endif
	auto time = std::chrono::duration_cast<std::chrono::microseconds>(high_resolution_clock::now() - startTime).count();
	debugf("Search took: %ums\n", (unsigned)time/1000);

	unsigned read_size = ETHASH_ACCESSES * ETHASH_MIX_BYTES;
#if defined(OPENCL) || defined(FULL)
	debugf(
		"hashrate: %8.2f Mh/s, bw: %8.2f GB/s\n",
		(double)hash_count * (1000*1000)/time / (1000*1000),
		(double)hash_count*read_size * (1000*1000)/time / (1024*1024*1024)
		);
#else
	debugf(
		"hashrate: %8.2f Kh/s, bw: %8.2f MB/s\n",
		(double)hash_count * (1000*1000)/time / (1000),
		(double)hash_count*read_size * (1000*1000)/time / (1024*1024)
		);
#endif

	free(cache_mem_buf);
#ifdef FULL
	free(full_mem_buf);
#endif

	return 0;
}
