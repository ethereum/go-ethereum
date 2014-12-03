
#pragma once

#include <vector>
#include <csetjmp>

//#include <libevm/ExtVMFace.h>

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

class Runtime
{
public:
	Runtime(u256 _gas, ExtVMFace& _ext, jmp_buf _jmpBuf, bool _outputLogs);

	Runtime(const Runtime&) = delete;
	void operator=(const Runtime&) = delete;

	RuntimeData* getDataPtr() { return &m_data; }

	StackImpl& getStack() { return m_stack; }
	MemoryImpl& getMemory() { return m_memory; }
	ExtVMFace& getExt() { return m_ext; }

	u256 getGas() const;
	bytes getReturnData() const;
	decltype(&jmp_buf{}[0]) getJmpBuf() { return m_data.jmpBuf; }
	bool outputLogs() const;

private:
	void set(RuntimeData::Index _index, u256 _value);

	/// @internal Must be the first element to asure Runtime* === RuntimeData*
	RuntimeData m_data;
	StackImpl m_stack;
	MemoryImpl m_memory;
	ExtVMFace& m_ext;
	bool m_outputLogs; ///< write LOG statements to console
};

}
}
}
