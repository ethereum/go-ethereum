
#pragma once

#include <vector>

#include <libevm/ExtVMFace.h>

#include "Utils.h"


#ifdef _MSC_VER
	#define EXPORT __declspec(dllexport)
#else
	#define EXPORT
#endif

namespace dev
{
namespace eth
{
namespace jit
{

using StackImpl = std::vector<i256>;
using MemoryImpl = dev::bytes;

class Runtime
{
public:
	Runtime(dev::u256 _gas, std::unique_ptr<dev::eth::ExtVMFace> _ext);
	~Runtime();

	Runtime(const Runtime&) = delete;
	void operator=(const Runtime&) = delete;

	static StackImpl& getStack();
	static MemoryImpl& getMemory();
	static dev::eth::ExtVMFace& getExt();
	static dev::u256 getGas();

private:
	StackImpl m_stack;
	MemoryImpl m_memory;
	std::unique_ptr<dev::eth::ExtVMFace> m_ext;
};

}
}
}
