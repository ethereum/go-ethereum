
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
		Caller,
		Origin,
		CallValue,
		CallDataSize,
		GasPrice,
		PrevHash,
		CoinBase,
		TimeStamp,
		Number,
		Difficulty,
		GasLimit,
		CodeSize,

		_size,

		ReturnDataOffset = CallValue,	// Reuse 2 fields for return data reference
		ReturnDataSize = CallDataSize
	};

	i256 elems[_size];
	byte const* callData;
	byte const* code;
	decltype(&jmp_buf{}[0]) jmpBuf;

	static llvm::StructType* getType();
};

using StackImpl = std::vector<i256>;
using MemoryImpl = bytes;

class Runtime
{
public:
	Runtime(u256 _gas, ExtVMFace& _ext, jmp_buf _jmpBuf);
	~Runtime();

	Runtime(const Runtime&) = delete;
	void operator=(const Runtime&) = delete;

	RuntimeData* getDataPtr() { return &m_data; }

	StackImpl& getStack() { return m_stack; }
	MemoryImpl& getMemory() { return m_memory; }
	ExtVMFace& getExt() { return m_ext; }

	u256 getGas() const;
	bytesConstRef getReturnData() const;
	decltype(&jmp_buf{}[0]) getJmpBuf() { return m_data.jmpBuf; }

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
	llvm::Value* get(Instruction _inst);
	llvm::Value* getGas();	// TODO: Remove
	llvm::Value* getCallData();
	llvm::Value* getCode();
	llvm::Value* getJmpBuf();
	void setGas(llvm::Value* _gas);

	void registerReturnData(llvm::Value* _index, llvm::Value* _size);

private:
	llvm::Value* getPtr(RuntimeData::Index _index);
	void set(RuntimeData::Index _index, llvm::Value* _value);

	llvm::GlobalVariable* m_dataPtr;
};

}
}
}
