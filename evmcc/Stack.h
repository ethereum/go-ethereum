
#pragma once

#include <vector>

#include <llvm/IR/IRBuilder.h>

namespace evmcc
{
class BasicBlock;

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
	BBStack() = default;
	BBStack(const BBStack&) = delete;
	void operator=(const BBStack&) = delete;

	/**
	 Changes current basic block (if any) with a new one with empty state.
	*/
	void setBasicBlock(BasicBlock& _newBlock);

	void push(llvm::Value* _value);
	llvm::Value* pop();

	/**
	 Duplicates _index'th value on stack.
	 */
	void dup(size_t _index);

	/**
	 Swaps _index'th value on stack with a value on stack top.
	 @param _index Index of value to be swaped. Cannot be 0.
	 */
	void swap(size_t _index);

private:
	BasicBlock* m_block = nullptr;		///< Current basic block
};


}