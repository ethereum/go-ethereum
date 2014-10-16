
#include "CompilerHelper.h"

#include <llvm/IR/Function.h>

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

}
}
}
