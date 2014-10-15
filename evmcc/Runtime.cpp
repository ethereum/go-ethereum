
#include "Runtime.h"

#include <libevm/VM.h>

#include "Type.h"

namespace evmcc
{

static Runtime* g_runtime;

extern "C"
{
EXPORT i256 gas;
}

Runtime::Runtime(dev::u256 _gas, std::unique_ptr<dev::eth::ExtVMFace> _ext):
	m_ext(std::move(_ext))
{
	assert(!g_runtime);
	g_runtime = this;
	gas = eth2llvm(_gas);
}

Runtime::~Runtime()
{
	g_runtime = nullptr;
}

StackImpl& Runtime::getStack()
{
	return g_runtime->m_stack;
}

MemoryImpl& Runtime::getMemory()
{
	return g_runtime->m_memory;
}

dev::eth::ExtVMFace& Runtime::getExt()
{
	return *g_runtime->m_ext;
}

dev::u256 Runtime::getGas()
{
	return llvm2eth(gas);
}

}