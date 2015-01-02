
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

	llvm::Value* sha3(llvm::Value* _inOff, llvm::Value* _inSize);
	MemoryRef getExtCode(llvm::Value* _addr);

	void log(llvm::Value* _memIdx, llvm::Value* _numBytes, std::array<llvm::Value*,4> const& _topics);

private:
	Memory& m_memoryMan;

	llvm::Value* m_args[2];
	llvm::Value* m_arg2;
	llvm::Value* m_arg3;
	llvm::Value* m_arg4;
	llvm::Value* m_arg5;
	llvm::Value* m_arg6;
	llvm::Value* m_arg7;
	llvm::Value* m_arg8;
	llvm::Value* m_size;
	llvm::Value* m_data = nullptr;
	llvm::Function* m_sload;
	llvm::Function* m_sstore;
	llvm::Function* m_calldataload;
	llvm::Function* m_balance = nullptr;
	llvm::Function* m_create;
	llvm::Function* m_call;
	llvm::Function* m_sha3;
	llvm::Function* m_getExtCode;
	llvm::Function* m_log;

	llvm::Function* getBalanceFunc();
};


}
}
}

