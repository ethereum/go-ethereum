
#include "BasicBlock.h"

#include <iostream>

#include <llvm/IR/CFG.h>
#include <llvm/IR/Function.h>
#include <llvm/IR/Instructions.h>
#include <llvm/IR/IRBuilder.h>
#include <llvm/Support/raw_os_ostream.h>

#include "Type.h"

namespace dev
{
namespace eth
{
namespace jit
{

const char* BasicBlock::NamePrefix = "Instr.";

BasicBlock::BasicBlock(bytes::const_iterator _begin, bytes::const_iterator _end, llvm::Function* _mainFunc, llvm::IRBuilder<>& _builder, bool isJumpDest) :
	m_begin(_begin),
	m_end(_end),
	// TODO: Add begin index to name
	m_llvmBB(llvm::BasicBlock::Create(_mainFunc->getContext(), NamePrefix, _mainFunc)),
	m_stack(*this),
	m_builder(_builder),
	m_isJumpDest(isJumpDest)
{}

BasicBlock::BasicBlock(std::string _name, llvm::Function* _mainFunc, llvm::IRBuilder<>& _builder, bool isJumpDest) :
	m_llvmBB(llvm::BasicBlock::Create(_mainFunc->getContext(), _name, _mainFunc)),
	m_stack(*this),
	m_builder(_builder),
	m_isJumpDest(isJumpDest)
{}

BasicBlock::LocalStack::LocalStack(BasicBlock& _owner) :
	m_bblock(_owner)
{}

void BasicBlock::LocalStack::push(llvm::Value* _value)
{
	m_bblock.m_currentStack.push_back(_value);
	m_bblock.m_tosOffset += 1;
}

llvm::Value* BasicBlock::LocalStack::pop()
{
	auto result = get(0);

	if (m_bblock.m_currentStack.size() > 0)
		m_bblock.m_currentStack.pop_back();

	m_bblock.m_tosOffset -= 1;
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

std::vector<llvm::Value*>::iterator BasicBlock::LocalStack::getItemIterator(size_t _index)
{
	auto& currentStack = m_bblock.m_currentStack;
	if (_index < currentStack.size())
		return currentStack.end() - _index - 1;

	// Need to map more elements from the EVM stack
	auto nNewItems = 1 + _index - currentStack.size();
	currentStack.insert(currentStack.begin(), nNewItems, nullptr);

	return currentStack.end() - _index - 1;
}

llvm::Value* BasicBlock::LocalStack::get(size_t _index)
{
	auto& initialStack = m_bblock.m_initialStack;
	auto itemIter = getItemIterator(_index);

	if (*itemIter == nullptr)
	{
		// Need to fetch a new item from the EVM stack
		assert(static_cast<int>(_index) >= m_bblock.m_tosOffset);
		size_t initialIdx = _index - m_bblock.m_tosOffset;
		if (initialIdx >= initialStack.size())
		{
			auto nNewItems = 1 + initialIdx - initialStack.size();
			initialStack.insert(initialStack.end(), nNewItems, nullptr);
		}

		assert(initialStack[initialIdx] == nullptr);
		// Create a dummy value.
		std::string name = "get_" + std::to_string(_index);
		initialStack[initialIdx] = m_bblock.m_builder.CreatePHI(Type::Word, 0, std::move(name));
		*itemIter = initialStack[initialIdx];
	}

	return *itemIter;
}

void BasicBlock::LocalStack::set(size_t _index, llvm::Value* _word)
{
	auto itemIter = getItemIterator(_index);
	*itemIter = _word;
}





void BasicBlock::synchronizeLocalStack(Stack& _evmStack)
{
	auto blockTerminator = m_llvmBB->getTerminator();
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
	for (; currIter < endIter; ++currIter)
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

		llvm::PHINode* phi = llvm::cast<llvm::PHINode>(val);
		// Insert call to get() just before the PHI node and replace
		// the uses of PHI with the uses of this new instruction.
		m_builder.SetInsertPoint(phi);
		auto newVal = _evmStack.get(idx); // OPT: Value may be never user but we need to check stack heigth
		                                  //      It is probably a good idea to keep heigth as a local variable accesible by LLVM directly
		phi->replaceAllUsesWith(newVal);
		phi->eraseFromParent();
	}

	// Reset the stack
	m_initialStack.erase(m_initialStack.begin(), m_initialStack.end());
	m_currentStack.erase(m_currentStack.begin(), m_currentStack.end());
	m_tosOffset = 0;
}

void BasicBlock::linkLocalStacks(std::vector<BasicBlock*> basicBlocks, llvm::IRBuilder<>& _builder)
{
	struct BBInfo
	{
		BasicBlock& bblock;
		std::vector<BBInfo*> predecessors;
		size_t inputItems;
		size_t outputItems;
		std::vector<llvm::PHINode*> phisToRewrite;

		BBInfo(BasicBlock& _bblock) :
			bblock(_bblock),
			predecessors(),
			inputItems(0),
			outputItems(0)
		{
			auto& initialStack = bblock.m_initialStack;
			for (auto it = initialStack.begin();
				 it != initialStack.end() && *it != nullptr;
				 ++it, ++inputItems);

			//if (bblock.localStack().m_tosOffset > 0)
			//	outputItems = bblock.localStack().m_tosOffset;
			auto& exitStack = bblock.m_currentStack;
			for (auto it = exitStack.rbegin();
				 it != exitStack.rend() && *it != nullptr;
				 ++it, ++outputItems);
		}
	};

	std::map<llvm::BasicBlock*, BBInfo> cfg;

	// Create nodes in cfg
	for (auto bb : basicBlocks)
		cfg.emplace(bb->llvm(), *bb);

	// Create edges in cfg: for each bb info fill the list
	// of predecessor infos.
	for (auto& pair : cfg)
	{
		auto bb = pair.first;
		auto& info = pair.second;

		for (auto predIt = llvm::pred_begin(bb); predIt != llvm::pred_end(bb); ++predIt)
		{
			auto predInfoEntry = cfg.find(*predIt);
			if (predInfoEntry != cfg.end())
				info.predecessors.push_back(&predInfoEntry->second);
		}
	}

	// Iteratively compute inputs and outputs of each block, until reaching fixpoint.
	bool valuesChanged = true;
	while (valuesChanged)
	{
		if (getenv("EVMCC_DEBUG_BLOCKS"))
		{
			for (auto& pair : cfg)
				std::cerr << pair.second.bblock.llvm()->getName().str()
						  << ": in " << pair.second.inputItems
						  << ", out " << pair.second.outputItems
						  << "\n";
		}

		valuesChanged = false;
		for (auto& pair : cfg)
		{
			auto& info = pair.second;

			if (info.predecessors.empty())
				info.inputItems = 0; // no consequences for other blocks, so leave valuesChanged false

			for (auto predInfo : info.predecessors)
			{
				if (predInfo->outputItems < info.inputItems)
				{
					info.inputItems = predInfo->outputItems;
					valuesChanged = true;
				}
				else if (predInfo->outputItems > info.inputItems)
				{
					predInfo->outputItems = info.inputItems;
					valuesChanged = true;
				}
			}
		}
	}

	// Propagate values between blocks.
	for (auto& entry : cfg)
	{
		auto& info = entry.second;
		auto& bblock = info.bblock;

		llvm::BasicBlock::iterator fstNonPhi(bblock.llvm()->getFirstNonPHI());
		auto phiIter = bblock.m_initialStack.begin();
		for (size_t index = 0; index < info.inputItems; ++index, ++phiIter)
		{
			assert(llvm::isa<llvm::PHINode>(*phiIter));
			auto phi = llvm::cast<llvm::PHINode>(*phiIter);

			for (auto predIt : info.predecessors)
			{
				auto& predExitStack = predIt->bblock.m_currentStack;
				auto value = *(predExitStack.end() - 1 - index);
				phi->addIncoming(value, predIt->bblock.llvm());
			}

			// Move phi to the front
			if (llvm::BasicBlock::iterator(phi) != bblock.llvm()->begin())
			{
				phi->removeFromParent();
				_builder.SetInsertPoint(bblock.llvm(), bblock.llvm()->begin());
				_builder.Insert(phi);
			}
		}

		// The items pulled directly from predecessors block must be removed
		// from the list of items that has to be popped from the initial stack.
		auto& initialStack = bblock.m_initialStack;
		initialStack.erase(initialStack.begin(), initialStack.begin() + info.inputItems);
		// Initial stack shrinks, so the size difference grows:
		bblock.m_tosOffset += info.inputItems;
	}

	// We must account for the items that were pushed directly to successor
	// blocks and thus should not be on the list of items to be pushed onto
	// to EVM stack
	for (auto& entry : cfg)
	{
		auto& info = entry.second;
		auto& bblock = info.bblock;

		auto& exitStack = bblock.m_currentStack;
		exitStack.erase(exitStack.end() - info.outputItems, exitStack.end());
		bblock.m_tosOffset -= info.outputItems;
	}
}

void BasicBlock::dump()
{
	dump(std::cerr, false);
}

void BasicBlock::dump(std::ostream& _out, bool _dotOutput)
{
	llvm::raw_os_ostream out(_out);

	out << (_dotOutput ? "" : "Initial stack:\n");
	for (auto val : m_initialStack)
	{
		if (val == nullptr)
			out << "  ?";
		else if (llvm::isa<llvm::Instruction>(val))
			out << *val;
		else
			out << "  " << *val;

		out << (_dotOutput ? "\\l" : "\n");
	}

	out << (_dotOutput ? "| " : "Instructions:\n");
	for (auto ins = m_llvmBB->begin(); ins != m_llvmBB->end(); ++ins)
		out << *ins << (_dotOutput ? "\\l" : "\n");

	if (! _dotOutput)
		out << "Current stack (offset = " << m_tosOffset << "):\n";
	else
		out << "|";

	for (auto val = m_currentStack.rbegin(); val != m_currentStack.rend(); ++val)
	{
		if (*val == nullptr)
			out << "  ?";
		else if (llvm::isa<llvm::Instruction>(*val))
			out << **val;
		else
			out << "  " << **val;
		out << (_dotOutput ? "\\l" : "\n");
	}

	if (! _dotOutput)
		out << "  ...\n----------------------------------------\n";
}




}
}
}

