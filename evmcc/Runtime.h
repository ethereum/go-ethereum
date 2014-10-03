
#pragma once

#include <vector>

#include <libevm/ExtVMFace.h>

#include "Utils.h"

namespace evmcc
{

using StackImpl = std::vector<i256>;
using MemoryImpl = dev::bytes;

class Runtime
{
public:
	Runtime(std::unique_ptr<dev::eth::ExtVMFace> _ext);
	~Runtime();

	Runtime(const Runtime&) = delete;
	void operator=(const Runtime&) = delete;

	static StackImpl& getStack();
	static MemoryImpl& getMemory();
	static dev::eth::ExtVMFace& getExt();

private:
	StackImpl m_stack;
	MemoryImpl m_memory;
	std::unique_ptr<dev::eth::ExtVMFace> m_ext;
};

}