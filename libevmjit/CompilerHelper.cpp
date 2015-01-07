
#include "CompilerHelper.h"

#include <llvm/IR/Function.h>
#include <llvm/IR/Module.h>

#include "RuntimeManager.h"

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
	// TODO: Rename or change semantics of getMainFunction() function
	assert(m_builder.GetInsertBlock());
	auto mainFunc = m_builder.GetInsertBlock()->getParent();
	assert(mainFunc);
	if (mainFunc == &mainFunc->getParent()->getFunctionList().front())  // Main function is the first one in module
		return mainFunc;
	return nullptr;
}

llvm::CallInst* CompilerHelper::createCall(llvm::Function* _func, std::initializer_list<llvm::Value*> const& _args)
{
	return getBuilder().CreateCall(_func, {_args.begin(), _args.size()});
}


RuntimeHelper::RuntimeHelper(RuntimeManager& _runtimeManager):
	CompilerHelper(_runtimeManager.getBuilder()),
	m_runtimeManager(_runtimeManager)
{}

}
}
}
