
#pragma once

#include <llvm/IR/Value.h>

namespace evmcc
{
class BasicBlock;

/**
 Stack adapter for Basic Block
 
 Transforms stack to SSA: tracks values and their positions on the imaginary stack used inside a basic block.
 TODO: Integrate into BasicBlock class
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