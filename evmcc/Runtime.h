
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
using MemoryImpl = bytes;

class Runtime
{
public:
	Runtime(u256 _gas, std::unique_ptr<ExtVMFace> _ext);
	~Runtime();

	Runtime(const Runtime&) = delete;
	void operator=(const Runtime&) = delete;

	static StackImpl& getStack();
	static MemoryImpl& getMemory();
	static ExtVMFace& getExt();
	static u256 getGas();

private:
	StackImpl m_stack;
	MemoryImpl m_memory;
	std::unique_ptr<ExtVMFace> m_ext;
};

}
}
}
