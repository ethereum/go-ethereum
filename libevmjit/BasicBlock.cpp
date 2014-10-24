
#include "BasicBlock.h"

#include <iostream>

#include <boost/lexical_cast.hpp>

#include <llvm/IR/Function.h>
#include <llvm/IR/Instructions.h>
#include <llvm/IR/IRBuilder.h>

#include "Type.h"

namespace dev
{
namespace eth
{
namespace jit
{

const char* BasicBlock::NamePrefix = "Instr.";

BasicBlock::BasicBlock(ProgramCounter _beginInstIdx, ProgramCounter _endInstIdx, llvm::Function* _mainFunc, llvm::IRBuilder<>& _builder) :
	m_beginInstIdx(_beginInstIdx),
	m_endInstIdx(_endInstIdx),
	m_llvmBB(llvm::BasicBlock::Create(_mainFunc->getContext(), {NamePrefix, std::to_string(_beginInstIdx)}, _mainFunc)),
	m_stack(_builder)
{}

BasicBlock::BasicBlock(std::string _name, llvm::Function* _mainFunc, llvm::IRBuilder<>& _builder) :
	m_beginInstIdx(0),
	m_endInstIdx(0),
	m_llvmBB(llvm::BasicBlock::Create(_mainFunc->getContext(), _name, _mainFunc)),
	m_stack(_builder)
{}

BasicBlock::LocalStack::LocalStack(llvm::IRBuilder<>& _builder) :
	m_builder(_builder),
	m_initialStack(),
	m_currentStack(),
	m_tosOffset(0)
{}

void BasicBlock::LocalStack::push(llvm::Value* _value)
{
	m_currentStack.push_back(_value);
	m_tosOffset += 1;
}

llvm::Value* BasicBlock::LocalStack::pop()
{
	auto result = get(0);

	if (m_currentStack.size() > 0)
		m_currentStack.pop_back();

	m_tosOffset -= 1;
	return result;
}

/**
 *  Pushes a copy of _index-th element (tos is 0-th elem).
 */
void BasicBlock::LocalStack::dup(size_t _index)
{
	auto val = get(_index);
	push(val);
}

/**
 *  Swaps tos with _index-th element (tos is 0-th elem).
 *  _index must be > 0.
 */
void BasicBlock::LocalStack::swap(size_t _index)
{
	assert(_index > 0);
	auto val = get(_index);
	auto tos = get(0);
	set(_index, tos);
	set(0, val);
}

void BasicBlock::LocalStack::synchronize(Stack& _evmStack)
{
	auto blockTerminator = m_builder.GetInsertBlock()->getTerminator();
	assert(blockTerminator != nullptr);
	m_builder.SetInsertPoint(blockTerminator);

	auto currIter = m_currentStack.begin();
	auto endIter = m_currentStack.end();

	// Update (emit set()) changed values
	for (int idx = m_currentStack.size() - 1 - m_tosOffset;
		 currIter < endIter && idx >= 0;
		 ++currIter, --idx)
	{
		assert(static_cast<size_t>(idx) < m_initialStack.size());
		if (*currIter != m_initialStack[idx]) // value needs update
			_evmStack.set(static_cast<size_t>(idx), *currIter);
	}

	if (m_tosOffset < 0)
	{
		// Pop values
		_evmStack.pop(static_cast<size_t>(-m_tosOffset));
	}

	// Push new values
	for ( ; currIter < endIter; ++currIter)
	{
		assert(*currIter != nullptr);
		_evmStack.push(*currIter);
	}

	// Emit get() for all (used) values from the initial stack
	for (size_t idx = 0; idx < m_initialStack.size(); ++idx)
	{
		auto val = m_initialStack[idx];
		if (val == nullptr)
			continue;

		assert(llvm::isa<llvm::PHINode>(val));
		llvm::PHINode* phi = llvm::cast<llvm::PHINode>(val);
		if (! phi->use_empty())
		{
			// Insert call to get() just before the PHI node and replace
			// the uses of PHI with the uses of this new instruction.
			m_builder.SetInsertPoint(phi);
			auto newVal = _evmStack.get(idx);
			phi->replaceAllUsesWith(newVal);
		}
		phi->eraseFromParent();
	}

	// Reset the stack
	m_initialStack.erase(m_initialStack.begin(), m_initialStack.end());
	m_currentStack.erase(m_currentStack.begin(), m_currentStack.end());
	m_tosOffset = 0;
}

std::vector<llvm::Value*>::iterator BasicBlock::LocalStack::getItemIterator(size_t _index)
{
	if (_index < m_currentStack.size())
		return m_currentStack.end() - _index - 1;

	// Need to map more elements from the EVM stack
	auto nNewItems = 1 + _index - m_currentStack.size();
	m_currentStack.insert(m_currentStack.begin(), nNewItems, nullptr);

	return m_currentStack.end() - _index - 1;
}

llvm::Value* BasicBlock::LocalStack::get(size_t _index)
{
	auto itemIter = getItemIterator(_index);

	if (*itemIter == nullptr)
	{
		// Need to fetch a new item from the EVM stack
		assert(static_cast<int>(_index) >= m_tosOffset);
		size_t initialIdx = _index - m_tosOffset;
		if (initialIdx >= m_initialStack.size())
		{
			auto nNewItems = 1 + initialIdx - m_initialStack.size();
			m_initialStack.insert(m_initialStack.end(), nNewItems, nullptr);
		}

		assert(m_initialStack[initialIdx] == nullptr);
		// Create a dummy value.
		std::string name = "get_" + boost::lexical_cast<std::string>(_index);
		m_initialStack[initialIdx] = m_builder.CreatePHI(Type::i256, 0, name);
		*itemIter = m_initialStack[initialIdx];
	}

	return *itemIter;
}

void BasicBlock::LocalStack::set(size_t _index, llvm::Value* _word)
{
	auto itemIter = getItemIterator(_index);
	*itemIter = _word;
}


void BasicBlock::dump()
{
	std::cerr << "Initial stack:\n";
	for (auto val : m_stack.m_initialStack)
	{
		if (val == nullptr)
			std::cerr << "  ?\n";
		else if (llvm::isa<llvm::Instruction>(val))
			val->dump();
		else
		{
			std::cerr << "  ";
			val->dump();
		}
	}
	std::cerr << "  ...\n";

    std::cerr << "Instructions:\n";
	for (auto ins = m_llvmBB->begin(); ins != m_llvmBB->end(); ++ins)
		ins->dump();

	std::cerr << "Current stack (offset = "
			  << m_stack.m_tosOffset << "):\n";

	for (auto val = m_stack.m_currentStack.rbegin(); val != m_stack.m_currentStack.rend(); ++val)
	{
		if (*val == nullptr)
			std::cerr << "  ?\n";
		else if (llvm::isa<llvm::Instruction>(*val))
			(*val)->dump();
		else
		{
			std::cerr << "  ";
			(*val)->dump();
		}

	}
	std::cerr << "  ...\n----------------------------------------\n";
}




}
}
}

