
#pragma once

#include <vector>

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

/**
 Stack adapter for Basic Block
 
 Transforms stack to SSA: tracks values and their positions on the imaginary stack used inside a basic block.
 */
class BBStack
{
public:
	//BBStack(llvm::IRBuilder<>& _builder, llvm::Module* _module);

	void push(llvm::Value* _value);
	llvm::Value* pop();
	void dup(size_t _index);
	void swap(size_t _index);

private:
	std::vector<llvm::Value*> m_state;	///< Basic black state vector - current values and their positions
};


}