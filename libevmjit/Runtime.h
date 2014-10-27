
#pragma once

#include <vector>

#include <libevm/ExtVMFace.h>

#include "CompilerHelper.h"
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

struct RuntimeData
{
	enum Index: unsigned
	{
		Gas,
		Address,

		_size
	};

	i256 elems[_size];

	static llvm::StructType* getType();
};

using StackImpl = std::vector<i256>;
using MemoryImpl = bytes;

class Runtime
{
public:
	Runtime(u256 _gas, ExtVMFace& _ext);
	~Runtime();

	Runtime(const Runtime&) = delete;
	void operator=(const Runtime&) = delete;

	RuntimeData* getDataPtr() { return &m_data; }

	StackImpl& getStack() { return m_stack; }
	MemoryImpl& getMemory() { return m_memory; }
	static ExtVMFace& getExt();

	u256 getGas() const;
	bytesConstRef getReturnData() const;

private:
	void set(RuntimeData::Index _index, u256 _value);

	/// @internal Must be the first element to asure Runtime* === RuntimeData*
	RuntimeData m_data;
	StackImpl m_stack;
	MemoryImpl m_memory;
	ExtVMFace& m_ext;
};

class RuntimeManager: public CompilerHelper
{
public:
	RuntimeManager(llvm::IRBuilder<>& _builder);

	llvm::Value* getRuntimePtr();

	llvm::Value* get(RuntimeData::Index _index);
	llvm::Value* getGas();
	void setGas(llvm::Value* _gas);

private:
	llvm::GlobalVariable* m_dataPtr;
};

}
}
}
