
#pragma once

#include <llvm/IR/IRBuilder.h>

#include <libevm/ExtVMFace.h>

namespace evmcc
{



class Ext
{
public:
	Ext(llvm::IRBuilder<>& _builder);
	static void init(std::unique_ptr<dev::eth::ExtVMFace> _ext);

	llvm::Value* store(llvm::Value* _index);
	void setStore(llvm::Value* _index, llvm::Value* _value);

private:
	llvm::IRBuilder<>& m_builder;

	llvm::Value* m_args[2];
	llvm::Function* m_store;
	llvm::Function* m_setStore;
};
	

}