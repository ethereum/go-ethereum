
#include "CompilerHelper.h"

#include <llvm/IR/Function.h>

#include "Runtime.h"

namespace dev
{
namespace eth
{
namespace jit
{

CompilerHelper::CompilerHelper(llvm::IRBuilder<>& _builder) :
	m_builder(_builder)
{}

llvm::Module* CompilerHelper::getModule()
{
	assert(m_builder.GetInsertBlock());
	assert(m_builder.GetInsertBlock()->getParent()); // BB must be in a function
	return m_builder.GetInsertBlock()->getParent()->getParent();
}

llvm::Function* CompilerHelper::getMainFunction()
{
	assert(m_builder.GetInsertBlock());
	auto mainFunc = m_builder.GetInsertBlock()->getParent();
	assert(mainFunc);
	if (mainFunc->getName() == "main")
		return mainFunc;
	return nullptr;
}


RuntimeHelper::RuntimeHelper(RuntimeManager& _runtimeManager):
	CompilerHelper(_runtimeManager.getBuilder()),
	m_runtimeManager(_runtimeManager)
{}

}
}
}
