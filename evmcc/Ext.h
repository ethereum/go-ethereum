
#pragma once

#include <llvm/IR/IRBuilder.h>

#include <libevm/ExtVMFace.h>

namespace evmcc
{



class Ext
{
public:
	Ext(llvm::IRBuilder<>& _builder, llvm::Module* module);
	static void init(std::unique_ptr<dev::eth::ExtVMFace> _ext);

	llvm::Value* store(llvm::Value* _index);
	void setStore(llvm::Value* _index, llvm::Value* _value);

	llvm::Value* address();
	llvm::Value* caller();
	llvm::Value* origin();
	llvm::Value* callvalue();
	llvm::Value* calldatasize();
	llvm::Value* gasprice();
	llvm::Value* prevhash();
	llvm::Value* coinbase();
	llvm::Value* timestamp();
	llvm::Value* number();
	llvm::Value* difficulty();
	llvm::Value* gaslimit();

	llvm::Value* balance(llvm::Value* _address);
	llvm::Value* calldataload(llvm::Value* _index);
	llvm::Value* create(llvm::Value* _endowment, llvm::Value* _initOff, llvm::Value* _initSize);

private:
	llvm::Value* getDataElem(unsigned _index, const llvm::Twine& _name = "");

	llvm::Value* bswap(llvm::Value*);

private:
	llvm::IRBuilder<>& m_builder;

	llvm::Value* m_args[2];
	llvm::Value* m_arg2;
	llvm::Value* m_arg3;
	llvm::Value* m_data;
	llvm::Function* m_init;
	llvm::Function* m_store;
	llvm::Function* m_setStore;
	llvm::Function* m_calldataload;
	llvm::Function* m_balance;
	llvm::Function* m_create;
	llvm::Function* m_bswap;
};
	

}
