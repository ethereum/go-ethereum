#pragma once

#define __CL_ENABLE_EXCEPTIONS
#define CL_USE_DEPRECATED_OPENCL_2_0_APIS

#if defined(__clang__)
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wunused-parameter"
#include "cl.hpp"
#pragma clang diagnostic pop
#else
#include "cl.hpp"
#endif

#include <time.h>
#include <functional>
#include <libethash/ethash.h>

class ethash_cl_miner
{
public:
	struct search_hook
	{
		virtual ~search_hook(); // always a virtual destructor for a class with virtuals.

		// reports progress, return true to abort
		virtual bool found(uint64_t const* nonces, uint32_t count) = 0;
		virtual bool searched(uint64_t start_nonce, uint32_t count) = 0;
	};

public:
	ethash_cl_miner();

	bool init(uint64_t block_number, std::function<void(void*)> _fillDAG, unsigned workgroup_size = 64, unsigned _platformId = 0, unsigned _deviceId = 0);
	static std::string platform_info(unsigned _platformId = 0, unsigned _deviceId = 0);
	static unsigned get_num_devices(unsigned _platformId = 0);


	void finish();
	void hash(uint8_t* ret, uint8_t const* header, uint64_t nonce, unsigned count);
	void search(uint8_t const* header, uint64_t target, search_hook& hook);

private:
	enum { c_max_search_results = 63, c_num_buffers = 2, c_hash_batch_size = 1024, c_search_batch_size = 1024*256 };

	uint64_t m_fullSize;
	cl::Context m_context;
	cl::CommandQueue m_queue;
	cl::Kernel m_hash_kernel;
	cl::Kernel m_search_kernel;
	cl::Buffer m_dag;
	cl::Buffer m_header;
	cl::Buffer m_hash_buf[c_num_buffers];
	cl::Buffer m_search_buf[c_num_buffers];
	unsigned m_workgroup_size;
	bool m_opencl_1_1;
};
