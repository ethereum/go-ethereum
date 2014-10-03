
#include "Runtime.h"

namespace evmcc
{

static Runtime* g_runtime;

Runtime::Runtime(std::unique_ptr<dev::eth::ExtVMFace> _ext)
	: m_ext(std::move(_ext))
{
	assert(!g_runtime);
	g_runtime = this;
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

}