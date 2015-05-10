// author Tim Hughes <tim@twistedfury.com>
// Tested on Radeon HD 7850
// Hashrate: 15940347 hashes/s
// Bandwidth: 124533 MB/s
// search kernel should fit in <= 84 VGPRS (3 wavefronts)

#define THREADS_PER_HASH (128 / 16)
#define HASHES_PER_LOOP (GROUP_SIZE / THREADS_PER_HASH)

#define FNV_PRIME	0x01000193

__constant uint2 const Keccak_f1600_RC[24] = {
	(uint2)(0x00000001, 0x00000000),
	(uint2)(0x00008082, 0x00000000),
	(uint2)(0x0000808a, 0x80000000),
	(uint2)(0x80008000, 0x80000000),
	(uint2)(0x0000808b, 0x00000000),
	(uint2)(0x80000001, 0x00000000),
	(uint2)(0x80008081, 0x80000000),
	(uint2)(0x00008009, 0x80000000),
	(uint2)(0x0000008a, 0x00000000),
	(uint2)(0x00000088, 0x00000000),
	(uint2)(0x80008009, 0x00000000),
	(uint2)(0x8000000a, 0x00000000),
	(uint2)(0x8000808b, 0x00000000),
	(uint2)(0x0000008b, 0x80000000),
	(uint2)(0x00008089, 0x80000000),
	(uint2)(0x00008003, 0x80000000),
	(uint2)(0x00008002, 0x80000000),
	(uint2)(0x00000080, 0x80000000),
	(uint2)(0x0000800a, 0x00000000),
	(uint2)(0x8000000a, 0x80000000),
	(uint2)(0x80008081, 0x80000000),
	(uint2)(0x00008080, 0x80000000),
	(uint2)(0x80000001, 0x00000000),
	(uint2)(0x80008008, 0x80000000),
};

void keccak_f1600_round(uint2* a, uint r, uint out_size)
{
   #if !__ENDIAN_LITTLE__
	for (uint i = 0; i != 25; ++i)
		a[i] = a[i].yx;
   #endif

	uint2 b[25];
	uint2 t;

	// Theta
	b[0] = a[0] ^ a[5] ^ a[10] ^ a[15] ^ a[20];
	b[1] = a[1] ^ a[6] ^ a[11] ^ a[16] ^ a[21];
	b[2] = a[2] ^ a[7] ^ a[12] ^ a[17] ^ a[22];
	b[3] = a[3] ^ a[8] ^ a[13] ^ a[18] ^ a[23];
	b[4] = a[4] ^ a[9] ^ a[14] ^ a[19] ^ a[24];
	t = b[4] ^ (uint2)(b[1].x << 1 | b[1].y >> 31, b[1].y << 1 | b[1].x >> 31);
	a[0] ^= t;
	a[5] ^= t;
	a[10] ^= t;
	a[15] ^= t;
	a[20] ^= t;
	t = b[0] ^ (uint2)(b[2].x << 1 | b[2].y >> 31, b[2].y << 1 | b[2].x >> 31);
	a[1] ^= t;
	a[6] ^= t;
	a[11] ^= t;
	a[16] ^= t;
	a[21] ^= t;
	t = b[1] ^ (uint2)(b[3].x << 1 | b[3].y >> 31, b[3].y << 1 | b[3].x >> 31);
	a[2] ^= t;
	a[7] ^= t;
	a[12] ^= t;
	a[17] ^= t;
	a[22] ^= t;
	t = b[2] ^ (uint2)(b[4].x << 1 | b[4].y >> 31, b[4].y << 1 | b[4].x >> 31);
	a[3] ^= t;
	a[8] ^= t;
	a[13] ^= t;
	a[18] ^= t;
	a[23] ^= t;
	t = b[3] ^ (uint2)(b[0].x << 1 | b[0].y >> 31, b[0].y << 1 | b[0].x >> 31);
	a[4] ^= t;
	a[9] ^= t;
	a[14] ^= t;
	a[19] ^= t;
	a[24] ^= t;

	// Rho Pi
	b[0] = a[0];
	b[10] = (uint2)(a[1].x << 1 | a[1].y >> 31, a[1].y << 1 | a[1].x >> 31);
	b[7] = (uint2)(a[10].x << 3 | a[10].y >> 29, a[10].y << 3 | a[10].x >> 29);
	b[11] = (uint2)(a[7].x << 6 | a[7].y >> 26, a[7].y << 6 | a[7].x >> 26);
	b[17] = (uint2)(a[11].x << 10 | a[11].y >> 22, a[11].y << 10 | a[11].x >> 22);
	b[18] = (uint2)(a[17].x << 15 | a[17].y >> 17, a[17].y << 15 | a[17].x >> 17);
	b[3] = (uint2)(a[18].x << 21 | a[18].y >> 11, a[18].y << 21 | a[18].x >> 11);
	b[5] = (uint2)(a[3].x << 28 | a[3].y >> 4, a[3].y << 28 | a[3].x >> 4);
	b[16] = (uint2)(a[5].y << 4 | a[5].x >> 28, a[5].x << 4 | a[5].y >> 28);
	b[8] = (uint2)(a[16].y << 13 | a[16].x >> 19, a[16].x << 13 | a[16].y >> 19);
	b[21] = (uint2)(a[8].y << 23 | a[8].x >> 9, a[8].x << 23 | a[8].y >> 9);
	b[24] = (uint2)(a[21].x << 2 | a[21].y >> 30, a[21].y << 2 | a[21].x >> 30);
	b[4] = (uint2)(a[24].x << 14 | a[24].y >> 18, a[24].y << 14 | a[24].x >> 18);
	b[15] = (uint2)(a[4].x << 27 | a[4].y >> 5, a[4].y << 27 | a[4].x >> 5);
	b[23] = (uint2)(a[15].y << 9 | a[15].x >> 23, a[15].x << 9 | a[15].y >> 23);
	b[19] = (uint2)(a[23].y << 24 | a[23].x >> 8, a[23].x << 24 | a[23].y >> 8);
	b[13] = (uint2)(a[19].x << 8 | a[19].y >> 24, a[19].y << 8 | a[19].x >> 24);
	b[12] = (uint2)(a[13].x << 25 | a[13].y >> 7, a[13].y << 25 | a[13].x >> 7);
	b[2] = (uint2)(a[12].y << 11 | a[12].x >> 21, a[12].x << 11 | a[12].y >> 21);
	b[20] = (uint2)(a[2].y << 30 | a[2].x >> 2, a[2].x << 30 | a[2].y >> 2);
	b[14] = (uint2)(a[20].x << 18 | a[20].y >> 14, a[20].y << 18 | a[20].x >> 14);
	b[22] = (uint2)(a[14].y << 7 | a[14].x >> 25, a[14].x << 7 | a[14].y >> 25);
	b[9] = (uint2)(a[22].y << 29 | a[22].x >> 3, a[22].x << 29 | a[22].y >> 3);
	b[6] = (uint2)(a[9].x << 20 | a[9].y >> 12, a[9].y << 20 | a[9].x >> 12);
	b[1] = (uint2)(a[6].y << 12 | a[6].x >> 20, a[6].x << 12 | a[6].y >> 20);

	// Chi
	a[0] = bitselect(b[0] ^ b[2], b[0], b[1]);
	a[1] = bitselect(b[1] ^ b[3], b[1], b[2]);
	a[2] = bitselect(b[2] ^ b[4], b[2], b[3]);
	a[3] = bitselect(b[3] ^ b[0], b[3], b[4]);
	if (out_size >= 4)
	{
		a[4] = bitselect(b[4] ^ b[1], b[4], b[0]);
		a[5] = bitselect(b[5] ^ b[7], b[5], b[6]);
		a[6] = bitselect(b[6] ^ b[8], b[6], b[7]);
		a[7] = bitselect(b[7] ^ b[9], b[7], b[8]);
		a[8] = bitselect(b[8] ^ b[5], b[8], b[9]);
		if (out_size >= 8)
		{
			a[9] = bitselect(b[9] ^ b[6], b[9], b[5]);
			a[10] = bitselect(b[10] ^ b[12], b[10], b[11]);
			a[11] = bitselect(b[11] ^ b[13], b[11], b[12]);
			a[12] = bitselect(b[12] ^ b[14], b[12], b[13]);
			a[13] = bitselect(b[13] ^ b[10], b[13], b[14]);
			a[14] = bitselect(b[14] ^ b[11], b[14], b[10]);
			a[15] = bitselect(b[15] ^ b[17], b[15], b[16]);
			a[16] = bitselect(b[16] ^ b[18], b[16], b[17]);
			a[17] = bitselect(b[17] ^ b[19], b[17], b[18]);
			a[18] = bitselect(b[18] ^ b[15], b[18], b[19]);
			a[19] = bitselect(b[19] ^ b[16], b[19], b[15]);
			a[20] = bitselect(b[20] ^ b[22], b[20], b[21]);
			a[21] = bitselect(b[21] ^ b[23], b[21], b[22]);
			a[22] = bitselect(b[22] ^ b[24], b[22], b[23]);
			a[23] = bitselect(b[23] ^ b[20], b[23], b[24]);
			a[24] = bitselect(b[24] ^ b[21], b[24], b[20]);
		}
	}

	// Iota
	a[0] ^= Keccak_f1600_RC[r];

   #if !__ENDIAN_LITTLE__
	for (uint i = 0; i != 25; ++i)
		a[i] = a[i].yx;
   #endif
}

void keccak_f1600_no_absorb(ulong* a, uint in_size, uint out_size, uint isolate)
{
	for (uint i = in_size; i != 25; ++i)
	{
		a[i] = 0;
	}
#if __ENDIAN_LITTLE__
	a[in_size] ^= 0x0000000000000001;
	a[24-out_size*2] ^= 0x8000000000000000;
#else
	a[in_size] ^= 0x0100000000000000;
	a[24-out_size*2] ^= 0x0000000000000080;
#endif

	// Originally I unrolled the first and last rounds to interface
	// better with surrounding code, however I haven't done this
	// without causing the AMD compiler to blow up the VGPR usage.
	uint r = 0;
	do
	{
		// This dynamic branch stops the AMD compiler unrolling the loop
		// and additionally saves about 33% of the VGPRs, enough to gain another
		// wavefront. Ideally we'd get 4 in flight, but 3 is the best I can
		// massage out of the compiler. It doesn't really seem to matter how
		// much we try and help the compiler save VGPRs because it seems to throw
		// that information away, hence the implementation of keccak here
		// doesn't bother.
		if (isolate) 
		{
			keccak_f1600_round((uint2*)a, r++, 25);
		}
	}
	while (r < 23);
	
	// final round optimised for digest size
	keccak_f1600_round((uint2*)a, r++, out_size);
}

#define copy(dst, src, count) for (uint i = 0; i != count; ++i) { (dst)[i] = (src)[i]; }

#define countof(x) (sizeof(x) / sizeof(x[0]))

uint fnv(uint x, uint y)
{
	return x * FNV_PRIME ^ y;
}

uint4 fnv4(uint4 x, uint4 y)
{
	return x * FNV_PRIME ^ y;
}

uint fnv_reduce(uint4 v)
{
	return fnv(fnv(fnv(v.x, v.y), v.z), v.w);
}

typedef union
{
	ulong ulongs[32 / sizeof(ulong)];
	uint uints[32 / sizeof(uint)];
} hash32_t;

typedef union
{
	ulong ulongs[64 / sizeof(ulong)];
	uint4 uint4s[64 / sizeof(uint4)];
} hash64_t;

typedef union
{
	uint uints[128 / sizeof(uint)];
	uint4 uint4s[128 / sizeof(uint4)];
} hash128_t;

hash64_t init_hash(__constant hash32_t const* header, ulong nonce, uint isolate)
{
	hash64_t init;
	uint const init_size = countof(init.ulongs);
	uint const hash_size = countof(header->ulongs);
	
	// sha3_512(header .. nonce)
	ulong state[25];
	copy(state, header->ulongs, hash_size);
	state[hash_size] = nonce;
	keccak_f1600_no_absorb(state, hash_size + 1, init_size, isolate);

	copy(init.ulongs, state, init_size);
	return init;
}

uint inner_loop(uint4 init, uint thread_id, __local uint* share, __global hash128_t const* g_dag, uint isolate)
{
	uint4 mix = init;

	// share init0
	if (thread_id == 0)
		*share = mix.x;
	barrier(CLK_LOCAL_MEM_FENCE);
	uint init0 = *share;

	uint a = 0;
	do
	{
		bool update_share = thread_id == (a/4) % THREADS_PER_HASH;

		#pragma unroll
		for (uint i = 0; i != 4; ++i)
		{
			if (update_share)
			{
				uint m[4] = { mix.x, mix.y, mix.z, mix.w };
				*share = fnv(init0 ^ (a+i), m[i]) % DAG_SIZE;
			}
			barrier(CLK_LOCAL_MEM_FENCE);

			mix = fnv4(mix, g_dag[*share].uint4s[thread_id]);
		}
	}
	while ((a += 4) != (ACCESSES & isolate));

	return fnv_reduce(mix);
}

hash32_t final_hash(hash64_t const* init, hash32_t const* mix, uint isolate)
{
	ulong state[25];

	hash32_t hash;
	uint const hash_size = countof(hash.ulongs);
	uint const init_size = countof(init->ulongs);
	uint const mix_size = countof(mix->ulongs);

	// keccak_256(keccak_512(header..nonce) .. mix);
	copy(state, init->ulongs, init_size);
	copy(state + init_size, mix->ulongs, mix_size);
	keccak_f1600_no_absorb(state, init_size+mix_size, hash_size, isolate);

	// copy out
	copy(hash.ulongs, state, hash_size);
	return hash;
}

hash32_t compute_hash_simple(
	__constant hash32_t const* g_header,
	__global hash128_t const* g_dag,
	ulong nonce,
	uint isolate
	)
{
	hash64_t init = init_hash(g_header, nonce, isolate);

	hash128_t mix;
	for (uint i = 0; i != countof(mix.uint4s); ++i)
	{
		mix.uint4s[i] = init.uint4s[i % countof(init.uint4s)];
	}
	
	uint mix_val = mix.uints[0];
	uint init0 = mix.uints[0];
	uint a = 0;
	do
	{
		uint pi = fnv(init0 ^ a, mix_val) % DAG_SIZE;
		uint n = (a+1) % countof(mix.uints);

		#pragma unroll
		for (uint i = 0; i != countof(mix.uints); ++i)
		{
			mix.uints[i] = fnv(mix.uints[i], g_dag[pi].uints[i]);
			mix_val = i == n ? mix.uints[i] : mix_val;
		}
	}
	while (++a != (ACCESSES & isolate));

	// reduce to output
	hash32_t fnv_mix;
	for (uint i = 0; i != countof(fnv_mix.uints); ++i)
	{
		fnv_mix.uints[i] = fnv_reduce(mix.uint4s[i]);
	}
	
	return final_hash(&init, &fnv_mix, isolate);
}

typedef union
{
	struct
	{
		hash64_t init;
		uint pad; // avoid lds bank conflicts
	};
	hash32_t mix;
} compute_hash_share;

hash32_t compute_hash(
	__local compute_hash_share* share,
	__constant hash32_t const* g_header,
	__global hash128_t const* g_dag,
	ulong nonce,
	uint isolate
	)
{
	uint const gid = get_global_id(0);

	// Compute one init hash per work item.
	hash64_t init = init_hash(g_header, nonce, isolate);

	// Threads work together in this phase in groups of 8.
	uint const thread_id = gid % THREADS_PER_HASH;
	uint const hash_id = (gid % GROUP_SIZE) / THREADS_PER_HASH;

	hash32_t mix;
	uint i = 0;
	do
	{
		// share init with other threads
		if (i == thread_id)
			share[hash_id].init = init;
		barrier(CLK_LOCAL_MEM_FENCE);

		uint4 thread_init = share[hash_id].init.uint4s[thread_id % (64 / sizeof(uint4))];
		barrier(CLK_LOCAL_MEM_FENCE);

		uint thread_mix = inner_loop(thread_init, thread_id, share[hash_id].mix.uints, g_dag, isolate);

		share[hash_id].mix.uints[thread_id] = thread_mix;
		barrier(CLK_LOCAL_MEM_FENCE);

		if (i == thread_id)
			mix = share[hash_id].mix;
		barrier(CLK_LOCAL_MEM_FENCE);
	}
	while (++i != (THREADS_PER_HASH & isolate));

	return final_hash(&init, &mix, isolate);
}

__attribute__((reqd_work_group_size(GROUP_SIZE, 1, 1)))
__kernel void ethash_hash_simple(
	__global hash32_t* g_hashes,
	__constant hash32_t const* g_header,
	__global hash128_t const* g_dag,
	ulong start_nonce,
	uint isolate
	)
{
	uint const gid = get_global_id(0);
	g_hashes[gid] = compute_hash_simple(g_header, g_dag, start_nonce + gid, isolate);
}

__attribute__((reqd_work_group_size(GROUP_SIZE, 1, 1)))
__kernel void ethash_search_simple(
	__global volatile uint* restrict g_output,
	__constant hash32_t const* g_header,
	__global hash128_t const* g_dag,
	ulong start_nonce,
	ulong target,
	uint isolate
	)
{
	uint const gid = get_global_id(0);
	hash32_t hash = compute_hash_simple(g_header, g_dag, start_nonce + gid, isolate);
	if (as_ulong(as_uchar8(hash.ulongs[0]).s76543210) < target)
	{
		uint slot = min(MAX_OUTPUTS, atomic_inc(&g_output[0]) + 1);
		g_output[slot] = gid;
	}
}

__attribute__((reqd_work_group_size(GROUP_SIZE, 1, 1)))
__kernel void ethash_hash(
	__global hash32_t* g_hashes,
	__constant hash32_t const* g_header,
	__global hash128_t const* g_dag,
	ulong start_nonce,
	uint isolate
	)
{
	__local compute_hash_share share[HASHES_PER_LOOP];

	uint const gid = get_global_id(0);
	g_hashes[gid] = compute_hash(share, g_header, g_dag, start_nonce + gid, isolate);
}

__attribute__((reqd_work_group_size(GROUP_SIZE, 1, 1)))
__kernel void ethash_search(
	__global volatile uint* restrict g_output,
	__constant hash32_t const* g_header,
	__global hash128_t const* g_dag,
	ulong start_nonce,
	ulong target,
	uint isolate
	)
{
	__local compute_hash_share share[HASHES_PER_LOOP];

	uint const gid = get_global_id(0);
	hash32_t hash = compute_hash(share, g_header, g_dag, start_nonce + gid, isolate);

	if (as_ulong(as_uchar8(hash.ulongs[0]).s76543210) < target)
	{
		uint slot = min(MAX_OUTPUTS, atomic_inc(&g_output[0]) + 1);
		g_output[slot] = gid;
	}
}
