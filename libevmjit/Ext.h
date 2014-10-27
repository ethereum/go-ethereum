
#pragma once

#include <libevm/ExtVMFace.h>

#include "CompilerHelper.h"

namespace dev
{
namespace eth
{
namespace jit
{

class Ext : public RuntimeHelper
{
public:
	Ext(RuntimeManager& _runtimeManager);

	llvm::Value* store(llvm::Value* _index);
	void setStore(llvm::Value* _index, llvm::Value* _value);

	llvm::Value* balance(llvm::Value* _address);
	void suicide(llvm::Value* _address);
	llvm::Value* calldataload(llvm::Value* _index);
	llvm::Value* create(llvm::Value* _endowment, llvm::Value* _initOff, llvm::Value* _initSize);
	llvm::Value* call(llvm::Value*& _gas, llvm::Value* _receiveAddress, llvm::Value* _value, llvm::Value* _inOff, llvm::Value* _inSize, llvm::Value* _outOff, llvm::Value* _outSize, llvm::Value* _codeAddress);

	llvm::Value* sha3(llvm::Value* _inOff, llvm::Value* _inSize);
	llvm::Value* exp(llvm::Value* _left, llvm::Value* _right);
	llvm::Value* codeAt(llvm::Value* _addr);
	llvm::Value* codesizeAt(llvm::Value* _addr);


private:
	llvm::Value* m_args[2];
	llvm::Value* m_arg2;
	llvm::Value* m_arg3;
	llvm::Value* m_arg4;
	llvm::Value* m_arg5;
	llvm::Value* m_arg6;
	llvm::Value* m_arg7;
	llvm::Value* m_arg8;
	llvm::Value* m_data;
	llvm::Function* m_store;
	llvm::Function* m_setStore;
	llvm::Function* m_calldataload;
	llvm::Function* m_balance;
	llvm::Function* m_suicide;
	llvm::Function* m_create;
	llvm::Function* m_call;
	llvm::Function* m_sha3;
	llvm::Function* m_exp;
	llvm::Function* m_codeAt;
	llvm::Function* m_codesizeAt;
};
	

}
}
}

