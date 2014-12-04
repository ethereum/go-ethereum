
#pragma once

#include <vector>

#include "Instruction.h"
#include "CompilerHelper.h"
#include "Utils.h"
#include "Type.h"
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
using JmpBufRef = decltype(&jmp_buf{}[0]);

/// VM Environment (ExtVM) opaque type
struct Env;

class Runtime
{
public:
	Runtime(RuntimeData* _data, Env* _env, JmpBufRef _jmpBuf);

	Runtime(const Runtime&) = delete;
	void operator=(const Runtime&) = delete;

	RuntimeData* getDataPtr() { return &m_data; } // FIXME: Remove

	StackImpl& getStack() { return m_stack; }
	MemoryImpl& getMemory() { return m_memory; }
	Env* getEnvPtr() { return &m_env; }

	u256 getGas() const;
	bytes getReturnData() const;
	JmpBufRef getJmpBuf() { return m_jmpBuf; }

private:
	RuntimeData& m_data;
	Env& m_env;
	JmpBufRef m_jmpBuf;
	StackImpl m_stack;
	MemoryImpl m_memory;
};

}
}
}
