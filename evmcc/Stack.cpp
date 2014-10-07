
#include "Stack.h"

#include <cassert>

#include <llvm/IR/Instructions.h>

#include "BasicBlock.h"

namespace evmcc
{

void BBStack::push(llvm::Value* _value)
{
	m_block->getState().push_back(_value);
}

llvm::Value* BBStack::pop()
{
	auto&& state = m_block->getState();
	if (state.empty())
	{
		// Create PHI node
		auto i256Ty = llvm::Type::getIntNTy(m_block->llvm()->getContext(), 256);
		auto llvmBB = m_block->llvm();
		if (llvmBB->empty())
			return llvm::PHINode::Create(i256Ty, 0, {}, m_block->llvm());
		return llvm::PHINode::Create(i256Ty, 0, {}, llvmBB->getFirstNonPHI());
	}

	auto top = state.back();
	state.pop_back();
	return top;
}

void BBStack::setBasicBlock(BasicBlock& _newBlock)
{
	// Current block keeps end state
	// Just update pointer to current block
	// New block should have empty state
	assert(_newBlock.getState().empty());
	m_block = &_newBlock;
}

void BBStack::dup(size_t _index)
{
	auto&& state = m_block->getState();
	auto value = *(state.rbegin() + _index);
	state.push_back(value);
}

void BBStack::swap(size_t _index)
{
	assert(_index != 0);
	auto&& state = m_block->getState();
	std::swap(*state.rbegin(), *(state.rbegin() + _index));
}

}
