
#include "Stack.h"

#include <vector>
#include <iostream>
#include <iomanip>
#include <cstdint>
#include <cassert>

#include <llvm/IR/Function.h>

#include "BasicBlock.h"
#include "Runtime.h"

#ifdef _MSC_VER
	#define EXPORT __declspec(dllexport)
#else
	#define EXPORT
#endif

namespace evmcc
{

BBStack::BBStack(llvm::IRBuilder<>& _builder, Stack& _extStack):
	m_extStack(_extStack),
	m_builder(_builder)
{}

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
		auto first = m_block->llvm()->getFirstNonPHI();
		auto llvmBB = m_block->llvm();
		if (llvmBB->getInstList().empty())
			return llvm::PHINode::Create(m_builder.getIntNTy(256), 0, {}, m_block->llvm());
		return llvm::PHINode::Create(m_builder.getIntNTy(256), 0, {}, llvmBB->getFirstNonPHI());
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

Stack::Stack(llvm::IRBuilder<>& _builder, llvm::Module* _module)
	: m_builder(_builder)
{
	// TODO: Clean up LLVM types
	auto stackPtrTy = m_builder.getInt8PtrTy();
	auto i256Ty = m_builder.getIntNTy(256);
	auto i256PtrTy = i256Ty->getPointerTo();
	auto voidTy = m_builder.getVoidTy();

	auto funcType = llvm::FunctionType::get(voidTy, i256PtrTy, false);
	m_stackPush = llvm::Function::Create(funcType,
		llvm::GlobalValue::LinkageTypes::ExternalLinkage, "evmccrt_stack_push", _module);

	m_stackPop = llvm::Function::Create(funcType,
		llvm::GlobalValue::LinkageTypes::ExternalLinkage, "evmccrt_stack_pop", _module);

	llvm::Type* getArgsTypes[] = {m_builder.getInt32Ty(), i256PtrTy};
	auto getFuncType = llvm::FunctionType::get(voidTy, getArgsTypes, false);
	m_stackGet = llvm::Function::Create(getFuncType,
		llvm::GlobalValue::LinkageTypes::ExternalLinkage, "evmccrt_stack_get", _module);

	m_stackSet = llvm::Function::Create(getFuncType,
		llvm::GlobalValue::LinkageTypes::ExternalLinkage, "evmccrt_stack_set", _module);

	m_stackVal = m_builder.CreateAlloca(i256Ty, nullptr, "stack.val");
}


void Stack::push(llvm::Value* _value)
{
	m_builder.CreateStore(_value, m_stackVal);	  // copy value to memory
	m_builder.CreateCall(m_stackPush, m_stackVal);
}


llvm::Value* Stack::pop()
{
	m_builder.CreateCall(m_stackPop, m_stackVal);
	return m_builder.CreateLoad(m_stackVal);
}


llvm::Value* Stack::get(uint32_t _index)
{
	llvm::Value* args[] = {m_builder.getInt32(_index), m_stackVal};
	m_builder.CreateCall(m_stackGet, args);
	return m_builder.CreateLoad(m_stackVal);
}


void Stack::set(uint32_t _index, llvm::Value* _value)
{
	m_builder.CreateStore(_value, m_stackVal);	  // copy value to memory
	llvm::Value* args[] = {m_builder.getInt32(_index), m_stackVal};
	m_builder.CreateCall(m_stackSet, args);
}


llvm::Value* Stack::top()
{
	return get(0);
}


void debugStack(const char* _op, const i256& _word, uint32_t _index = 0)
{
	std::cerr << "STACK " << std::setw(4) << std::setfill(' ') << _op
			  << " [" << std::setw(2) << std::setfill('0') << _index << "] "
			  << std::dec << _word.a
			  << " HEX: " << std::hex;
	if (_word.b || _word.c || _word.d)
	{
		std::cerr << std::setw(16) << _word.d << " "
				  << std::setw(16) << _word.c << " "
				  << std::setw(16) << _word.b << " ";
	}
	std::cerr << std::setw(16) << _word.a << "\n";
}

}

extern "C"
{
	using namespace evmcc;

EXPORT void evmccrt_stack_push(i256* _word)
{
	//debugStack("push", *_word);
	Runtime::getStack().push_back(*_word);
}

EXPORT void evmccrt_stack_pop(i256* _outWord)
{
	assert(!Runtime::getStack().empty());
	auto word = &Runtime::getStack().back();
	//debugStack("pop", *word);
	Runtime::getStack().pop_back();
	*_outWord = *word;
}

EXPORT void evmccrt_stack_get(uint32_t _index, i256* _outWord)
{
	assert(_index < Runtime::getStack().size());
	auto word = Runtime::getStack().rbegin() + _index;
	//debugStack("get", *word, _index);
	*_outWord = *word;
}

EXPORT void evmccrt_stack_set(uint32_t _index, i256* _word)
{
	assert(_index < Runtime::getStack().size());
	*(Runtime::getStack().rbegin() + _index) = *_word;
	//debugStack("set", *_word, _index);
}

}	// extern "C"
