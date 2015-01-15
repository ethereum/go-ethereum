
#pragma once

#include <array>
#include "CompilerHelper.h"

namespace dev
{
namespace eth
{
namespace jit
{
	class Memory;

struct MemoryRef
{
	llvm::Value* ptr;
	llvm::Value* size;
};

template<typename _EnumT>
struct sizeOf
{
	static const size_t value = static_cast<size_t>(_EnumT::_size);
};

enum class EnvFunc
{
	sload,
	sstore,
	sha3,
	balance,
	create,
	call,
	log,
	blockhash,
	extcode,
	calldataload,  // Helper function, not client Env interface

	_size
};

class Ext : public RuntimeHelper
{
public:
	Ext(RuntimeManager& _runtimeManager, Memory& _memoryMan);

	llvm::Value* sload(llvm::Value* _index);
	void sstore(llvm::Value* _index, llvm::Value* _value);

	llvm::Value* balance(llvm::Value* _address);
	llvm::Value* calldataload(llvm::Value* _index);
	llvm::Value* create(llvm::Value*& _gas, llvm::Value* _endowment, llvm::Value* _initOff, llvm::Value* _initSize);
	llvm::Value* call(llvm::Value*& _gas, llvm::Value* _receiveAddress, llvm::Value* _value, llvm::Value* _inOff, llvm::Value* _inSize, llvm::Value* _outOff, llvm::Value* _outSize, llvm::Value* _codeAddress);
	llvm::Value* blockhash(llvm::Value* _number);

	llvm::Value* sha3(llvm::Value* _inOff, llvm::Value* _inSize);
	MemoryRef extcode(llvm::Value* _addr);

	void log(llvm::Value* _memIdx, llvm::Value* _numBytes, std::array<llvm::Value*,4> const& _topics);

private:
	Memory& m_memoryMan;

	llvm::Value* m_size;
	llvm::Value* m_data = nullptr;

	std::array<llvm::Function*, sizeOf<EnvFunc>::value> m_funcs;
	std::array<llvm::Value*, 8> m_argAllocas;
	size_t m_argCounter = 0;

	llvm::CallInst* createCall(EnvFunc _funcId, std::initializer_list<llvm::Value*> const& _args);
	llvm::Value* getArgAlloca();
	llvm::Value* byPtr(llvm::Value* _value);
};


}
}
}

