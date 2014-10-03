
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
	void set(uint32_t _index, llvm::Value* _value);

private:
	llvm::IRBuilder<>& m_builder;
	llvm::Value* m_stackVal;
	llvm::Function* m_stackPush;
	llvm::Function* m_stackPop;
	llvm::Function* m_stackGet;
	llvm::Function* m_stackSet;
};

}