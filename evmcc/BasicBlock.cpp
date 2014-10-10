
#include "BasicBlock.h"

#include <llvm/IR/Function.h>
#include <llvm/IR/Instructions.h>

#include "Type.h"

namespace evmcc
{

const char* BasicBlock::NamePrefix = "Instr.";

BasicBlock::BasicBlock(ProgramCounter _beginInstIdx, ProgramCounter _endInstIdx, llvm::Function* _mainFunc) :
	m_beginInstIdx(_beginInstIdx),
	m_endInstIdx(_endInstIdx),
	m_llvmBB(llvm::BasicBlock::Create(_mainFunc->getContext(), {NamePrefix, std::to_string(_beginInstIdx)}, _mainFunc)),
	m_stack(m_llvmBB)
{}

BasicBlock::BasicBlock(std::string _name, llvm::Function* _mainFunc) :
	m_beginInstIdx(0),
	m_endInstIdx(0),
	m_llvmBB(llvm::BasicBlock::Create(_mainFunc->getContext(), _name, _mainFunc)),
	m_stack(m_llvmBB)
{}


void BasicBlock::Stack::push(llvm::Value* _value)
{
	m_backend.push_back(_value);
}

llvm::Value* BasicBlock::Stack::pop()
{
	if (m_backend.empty())
	{
		// Create PHI node
		if (m_llvmBB->empty())
			return llvm::PHINode::Create(Type::i256, 0, {}, m_llvmBB);
		return llvm::PHINode::Create(Type::i256, 0, {}, m_llvmBB->getFirstNonPHI());
	}

	auto top = m_backend.back();
	m_backend.pop_back();
	return top;
}

void BasicBlock::Stack::dup(size_t _index)
{
	m_backend.push_back(get(_index));
}

void BasicBlock::Stack::swap(size_t _index)
{
	assert(_index != 0);
	std::swap(get(0), get(_index));
}

}
