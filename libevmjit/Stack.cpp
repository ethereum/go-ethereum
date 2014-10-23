#include "Stack.h"
#include "Runtime.h"
#include "Type.h"

#include <csetjmp>

#include <llvm/IR/Function.h>
#include <llvm/IR/TypeBuilder.h>

namespace dev
{
namespace eth
{
namespace jit
{

Stack::Stack(llvm::IRBuilder<>& _builder)
	: CompilerHelper(_builder)
{
	m_arg = m_builder.CreateAlloca(Type::i256, nullptr, "stack.arg");

	using namespace llvm;
	using Linkage = GlobalValue::LinkageTypes;

	auto module = getModule();
	m_push = Function::Create(FunctionType::get(m_builder.getVoidTy(), Type::WordPtr, false), Linkage::ExternalLinkage, "stack_push", module);
	m_pop = Function::Create(FunctionType::get(m_builder.getVoidTy(), Type::Size, false), Linkage::ExternalLinkage, "stack_pop", module);
	llvm::Type* getSetArgTypes[] = {Type::Size, Type::WordPtr};
	m_get = Function::Create(FunctionType::get(m_builder.getVoidTy(), ArrayRef<llvm::Type*>(getSetArgTypes), false), Linkage::ExternalLinkage, "stack_get", module);
	m_set = Function::Create(FunctionType::get(m_builder.getVoidTy(), ArrayRef<llvm::Type*>(getSetArgTypes), false), Linkage::ExternalLinkage, "stack_set", module);
}

Stack::~Stack()
{}

llvm::Value* Stack::get(size_t _index)
{
	m_builder.CreateCall2(m_get, llvm::ConstantInt::get(Type::Size, _index, false), m_arg);
	return m_builder.CreateLoad(m_arg);
}

void Stack::set(size_t _index, llvm::Value* _value)
{
	m_builder.CreateStore(_value, m_arg);
	m_builder.CreateCall2(m_set, llvm::ConstantInt::get(Type::Size, _index, false), m_arg);
}

void Stack::pop(size_t _count)
{
	m_builder.CreateCall(m_pop, llvm::ConstantInt::get(Type::Size, _count, false));
}

void Stack::push(llvm::Value* _value)
{
	m_builder.CreateStore(_value, m_arg);
	m_builder.CreateCall(m_push, m_arg);
}


size_t Stack::maxStackSize = 0;

}
}
}

extern "C"
{

using namespace dev::eth::jit;

extern std::jmp_buf* rt_jmpBuf;

EXPORT void stack_pop(uint64_t _count)
{
	auto& stack = Runtime::getStack();
	if (stack.size() < _count)
		longjmp(*rt_jmpBuf, static_cast<uint64_t>(ReturnCode::StackTooSmall));

	stack.erase(stack.end() - _count, stack.end());
}

EXPORT void stack_push(i256* _word)
{
	auto& stack = Runtime::getStack();
	stack.push_back(*_word);

	if (stack.size() > Stack::maxStackSize)
		Stack::maxStackSize = stack.size();
}

EXPORT void stack_get(uint64_t _index, i256* _ret)
{
	auto& stack = Runtime::getStack();
	// TODO: encode _index and stack size in the return code
	if (stack.size() <= _index)
		longjmp(*rt_jmpBuf, static_cast<uint64_t>(ReturnCode::StackTooSmall));

	*_ret = *(stack.rbegin() + _index);
}

EXPORT void stack_set(uint64_t _index, i256* _word)
{
	auto& stack = Runtime::getStack();
	// TODO: encode _index and stack size in the return code
	if (stack.size() <= _index)
		longjmp(*rt_jmpBuf, static_cast<uint64_t>(ReturnCode::StackTooSmall));

	*(stack.rbegin() + _index) = *_word;
}

} // extern "C"

