#include "Stack.h"
#include "Runtime.h"

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
	auto i256Ty = m_builder.getIntNTy(256);
	auto i256PtrTy = i256Ty->getPointerTo();

	m_arg = m_builder.CreateAlloca(i256Ty, nullptr, "stack.retVal");

	using namespace llvm;
	using Linkage = GlobalValue::LinkageTypes;

	auto module = getModule();
	m_push = Function::Create(FunctionType::get(m_builder.getVoidTy(), i256PtrTy, false), Linkage::ExternalLinkage, "stack_push", module);
	m_pop = Function::Create(FunctionType::get(m_builder.getVoidTy(), i256PtrTy, false), Linkage::ExternalLinkage, "stack_pop", module);
}

Stack::~Stack()
{}

llvm::Instruction* Stack::popWord()
{
	m_builder.CreateCall(m_pop, m_arg);
	return m_builder.CreateLoad(m_arg);
}

void Stack::pushWord(llvm::Value* _word)
{
	m_builder.CreateStore(_word, m_arg);
	m_builder.CreateCall(m_push, m_arg);
}

}
}
}

extern "C"
{

using namespace dev::eth::jit;

EXPORT void stack_pop(i256* _ret)
{
	auto& stack = Runtime::getStack();
	assert(stack.size() > 0);
	*_ret = stack.back();
	stack.pop_back();
}

EXPORT void stack_push(i256* _word)
{
	auto& stack = Runtime::getStack();
	stack.push_back(*_word);
}

} // extern "C"

