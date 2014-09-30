
#pragma once

#include <llvm/IR/IRBuilder.h>

namespace evmcc
{

class Stack
{
public:
	Stack(llvm::IRBuilder<>& _builder, llvm::Module* _module);

	void push(llvm::Value* _value);
	llvm::Value* pop();
	llvm::Value* top();
	llvm::Value* get(uint32_t _index);

private:
	llvm::IRBuilder<>& m_builder;
	llvm::Value* m_args[2];
	llvm::Function* m_stackPush;
	llvm::Function* m_stackPop;
	llvm::Function* m_stackGet;
};

}