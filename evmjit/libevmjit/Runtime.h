
#pragma once

#include <csetjmp>
#include "RuntimeData.h"

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
using jmp_buf_ref = decltype(&std::jmp_buf{}[0]);

class Runtime
{
public:
	Runtime(RuntimeData* _data, Env* _env);

	Runtime(const Runtime&) = delete;
	Runtime& operator=(const Runtime&) = delete;

	StackImpl& getStack() { return m_stack; }
	MemoryImpl& getMemory() { return m_memory; }
	Env* getEnvPtr() { return &m_env; }

	bytes_ref getReturnData() const;
	jmp_buf_ref getJmpBuf() { return m_jmpBuf; }

private:
	RuntimeData& m_data;		///< Pointer to data. Expected by compiled contract.
	Env& m_env;					///< Pointer to environment proxy. Expected by compiled contract.
	jmp_buf_ref m_currJmpBuf;	///< Pointer to jump buffer. Expected by compiled contract.
	byte* m_memoryData = nullptr;
	i256 m_memorySize;
	std::jmp_buf m_jmpBuf;
	StackImpl m_stack;
	MemoryImpl m_memory;
};

}
}
}
