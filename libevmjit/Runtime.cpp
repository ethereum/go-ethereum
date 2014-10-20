
#include "Runtime.h"

#include <libevm/VM.h>

#include "Type.h"

namespace dev
{
namespace eth
{
namespace jit
{

static Runtime* g_runtime;

extern "C"
{
EXPORT i256 gas;
}

Runtime::Runtime(u256 _gas, ExtVMFace& _ext):
	m_ext(_ext)
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

ExtVMFace& Runtime::getExt()
{
	return g_runtime->m_ext;
}

u256 Runtime::getGas()
{
	return llvm2eth(gas);
}

}
}
}
