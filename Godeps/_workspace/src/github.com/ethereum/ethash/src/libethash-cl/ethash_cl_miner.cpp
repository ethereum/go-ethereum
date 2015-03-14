/*
  This file is part of c-ethash.

  c-ethash is free software: you can redistribute it and/or modify
  it under the terms of the GNU General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  c-ethash is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU General Public License for more details.

  You should have received a copy of the GNU General Public License
  along with cpp-ethereum.  If not, see <http://www.gnu.org/licenses/>.
*/
/** @file ethash_cl_miner.cpp
* @author Tim Hughes <tim@twistedfury.com>
* @date 2015
*/


#define _CRT_SECURE_NO_WARNINGS

#include <assert.h>
#include <queue>
#include <vector>
#include "ethash_cl_miner.h"
#include "ethash_cl_miner_kernel.h"
#include <libethash/util.h>

#define ETHASH_BYTES 32

// workaround lame platforms
#if !CL_VERSION_1_2
#define CL_MAP_WRITE_INVALIDATE_REGION CL_MAP_WRITE
#define CL_MEM_HOST_READ_ONLY 0
#endif

#undef min
#undef max

static void add_definition(std::string& source, char const* id, unsigned value)
{
	char buf[256];
	sprintf(buf, "#define %s %uu\n", id, value);
	source.insert(source.begin(), buf, buf + strlen(buf));
}

ethash_cl_miner::ethash_cl_miner()
:	m_opencl_1_1()
{
}

bool ethash_cl_miner::init(ethash_params const& params, const uint8_t seed[32], unsigned workgroup_size)
{
	// store params
	m_params = params;

	// get all platforms
    std::vector<cl::Platform> platforms;
    cl::Platform::get(&platforms);
	if (platforms.empty())
	{
		debugf("No OpenCL platforms found.\n");
		return false;
	}

	// use default platform
	debugf("Using platform: %s\n", platforms[0].getInfo<CL_PLATFORM_NAME>().c_str());

    // get GPU device of the default platform
    std::vector<cl::Device> devices;
    platforms[0].getDevices(CL_DEVICE_TYPE_ALL, &devices);
    if (devices.empty())
	{
		debugf("No OpenCL devices found.\n");
		return false;
	}

	// use default device
	unsigned device_num = 0;
	cl::Device& device = devices[device_num];
	std::string device_version = device.getInfo<CL_DEVICE_VERSION>();
	debugf("Using device: %s (%s)\n", device.getInfo<CL_DEVICE_NAME>().c_str(),device_version.c_str());

	if (strncmp("OpenCL 1.0", device_version.c_str(), 10) == 0)
	{
		debugf("OpenCL 1.0 is not supported.\n");
		return false;
	}
	if (strncmp("OpenCL 1.1", device_version.c_str(), 10) == 0)
	{
		m_opencl_1_1 = true;
	}

	// create context
	m_context = cl::Context(std::vector<cl::Device>(&device, &device+1));
	m_queue = cl::CommandQueue(m_context, device);

	// use requested workgroup size, but we require multiple of 8
	m_workgroup_size = ((workgroup_size + 7) / 8) * 8;

	// patch source code
	std::string code(ETHASH_CL_MINER_KERNEL, ETHASH_CL_MINER_KERNEL + ETHASH_CL_MINER_KERNEL_SIZE);
	add_definition(code, "GROUP_SIZE", m_workgroup_size);
	add_definition(code, "DAG_SIZE", (unsigned)(params.full_size / MIX_BYTES));
	add_definition(code, "ACCESSES", ACCESSES);
	add_definition(code, "MAX_OUTPUTS", c_max_search_results);
	//debugf("%s", code.c_str());

	// create miner OpenCL program
	cl::Program::Sources sources;
	sources.push_back({code.c_str(), code.size()});

	cl::Program program(m_context, sources);
	try
	{
		program.build({device});
	}
	catch (cl::Error err)
	{
		debugf("%s\n", program.getBuildInfo<CL_PROGRAM_BUILD_LOG>(device).c_str());
		return false;
	}
	m_hash_kernel = cl::Kernel(program, "ethash_hash");
	m_search_kernel = cl::Kernel(program, "ethash_search");

	// create buffer for dag
	m_dag = cl::Buffer(m_context, CL_MEM_READ_ONLY, params.full_size);
	
	// create buffer for header
	m_header = cl::Buffer(m_context, CL_MEM_READ_ONLY, 32);

	// compute dag on CPU
	{
		void* cache_mem = malloc(params.cache_size + 63);
		ethash_cache cache;
		cache.mem = (void*)(((uintptr_t)cache_mem + 63) & ~63);
		ethash_mkcache(&cache, &params, seed);

		// if this throws then it's because we probably need to subdivide the dag uploads for compatibility
		void* dag_ptr = m_queue.enqueueMapBuffer(m_dag, true, m_opencl_1_1 ? CL_MAP_WRITE : CL_MAP_WRITE_INVALIDATE_REGION, 0, params.full_size);
		ethash_compute_full_data(dag_ptr, &params, &cache);
		m_queue.enqueueUnmapMemObject(m_dag, dag_ptr);

		free(cache_mem);
	}

	// create mining buffers
	for (unsigned i = 0; i != c_num_buffers; ++i)
	{
		m_hash_buf[i] = cl::Buffer(m_context, CL_MEM_WRITE_ONLY | (!m_opencl_1_1 ? CL_MEM_HOST_READ_ONLY : 0), 32*c_hash_batch_size);
		m_search_buf[i] = cl::Buffer(m_context, CL_MEM_WRITE_ONLY, (c_max_search_results + 1) * sizeof(uint32_t));
	}
	return true;
}

void ethash_cl_miner::hash(uint8_t* ret, uint8_t const* header, uint64_t nonce, unsigned count)
{
	struct pending_batch
	{
		unsigned base;
		unsigned count;
		unsigned buf;
	};
	std::queue<pending_batch> pending;
	
	// update header constant buffer
	m_queue.enqueueWriteBuffer(m_header, true, 0, 32, header);

	/*
	__kernel void ethash_combined_hash(
		__global hash32_t* g_hashes,
		__constant hash32_t const* g_header,
		__global hash128_t const* g_dag,
		ulong start_nonce,
		uint isolate
		)
	*/
	m_hash_kernel.setArg(1, m_header);
	m_hash_kernel.setArg(2, m_dag);
	m_hash_kernel.setArg(3, nonce);
	m_hash_kernel.setArg(4, ~0u); // have to pass this to stop the compile unrolling the loop

	unsigned buf = 0;
	for (unsigned i = 0; i < count || !pending.empty(); )
	{
		// how many this batch
		if (i < count)
		{
			unsigned const this_count = std::min(count - i, c_hash_batch_size);
			unsigned const batch_count = std::max(this_count, m_workgroup_size);

			// supply output hash buffer to kernel
			m_hash_kernel.setArg(0, m_hash_buf[buf]);

			// execute it!
			m_queue.enqueueNDRangeKernel(
				m_hash_kernel,
				cl::NullRange,
				cl::NDRange(batch_count),
				cl::NDRange(m_workgroup_size)
				);
			m_queue.flush();
		
			pending.push({i, this_count, buf});
			i += this_count;
			buf = (buf + 1) % c_num_buffers;
		}

		// read results
		if (i == count || pending.size() == c_num_buffers)
		{
			pending_batch const& batch = pending.front();

			// could use pinned host pointer instead, but this path isn't that important.
			uint8_t* hashes = (uint8_t*)m_queue.enqueueMapBuffer(m_hash_buf[batch.buf], true, CL_MAP_READ, 0, batch.count * ETHASH_BYTES);
			memcpy(ret + batch.base*ETHASH_BYTES, hashes, batch.count*ETHASH_BYTES);
			m_queue.enqueueUnmapMemObject(m_hash_buf[batch.buf], hashes);

			pending.pop();
		}
	}
}


void ethash_cl_miner::search(uint8_t const* header, uint64_t target, search_hook& hook)
{
	struct pending_batch
	{
		uint64_t start_nonce;
		unsigned buf;
	};
	std::queue<pending_batch> pending;

	static uint32_t const c_zero = 0;

	// update header constant buffer
	m_queue.enqueueWriteBuffer(m_header, false, 0, 32, header);
	for (unsigned i = 0; i != c_num_buffers; ++i)
	{
		m_queue.enqueueWriteBuffer(m_search_buf[i], false, 0, 4, &c_zero);
	}

#if CL_VERSION_1_2
	cl::Event pre_return_event;
	if (!m_opencl_1_1)
	{
		m_queue.enqueueBarrierWithWaitList(NULL, &pre_return_event);
	}
	else
#else
	{
		m_queue.finish();
	}
#endif

	/*
	__kernel void ethash_combined_search(
		__global hash32_t* g_hashes,			// 0
		__constant hash32_t const* g_header,	// 1
		__global hash128_t const* g_dag,		// 2
		ulong start_nonce,						// 3
		ulong target,							// 4
		uint isolate							// 5
	)
	*/
	m_search_kernel.setArg(1, m_header);
	m_search_kernel.setArg(2, m_dag);

	// pass these to stop the compiler unrolling the loops
	m_search_kernel.setArg(4, target);
	m_search_kernel.setArg(5, ~0u);


	unsigned buf = 0;
	for (uint64_t start_nonce = 0; ; start_nonce += c_search_batch_size)
	{
		// supply output buffer to kernel
		m_search_kernel.setArg(0, m_search_buf[buf]);
		m_search_kernel.setArg(3, start_nonce);

		// execute it!
		m_queue.enqueueNDRangeKernel(m_search_kernel, cl::NullRange, c_search_batch_size, m_workgroup_size);
		
		pending.push({start_nonce, buf});
		buf = (buf + 1) % c_num_buffers;

		// read results
		if (pending.size() == c_num_buffers)
		{
			pending_batch const& batch = pending.front();

			// could use pinned host pointer instead
			uint32_t* results = (uint32_t*)m_queue.enqueueMapBuffer(m_search_buf[batch.buf], true, CL_MAP_READ, 0, (1+c_max_search_results) * sizeof(uint32_t));
			unsigned num_found = std::min(results[0], c_max_search_results);

			uint64_t nonces[c_max_search_results];
			for (unsigned i = 0; i != num_found; ++i)
			{
				nonces[i] = batch.start_nonce + results[i+1];
			}
			
			m_queue.enqueueUnmapMemObject(m_search_buf[batch.buf], results);
			
			bool exit = num_found && hook.found(nonces, num_found);
			exit |= hook.searched(batch.start_nonce, c_search_batch_size); // always report searched before exit
			if (exit)
				break;

			pending.pop();
		}
	}

	// not safe to return until this is ready
#if CL_VERSION_1_2
	if (!m_opencl_1_1)
	{
		pre_return_event.wait();
	}
#endif
}

