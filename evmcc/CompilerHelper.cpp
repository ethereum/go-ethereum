
#include "CompilerHelper.h"

#include <llvm/IR/Function.h>

namespace dev
{
namespace eth
{
namespace jit
{

CompilerHelper::CompilerHelper(llvm::IRBuilder<>& _builder) :
	m_builder(_builder),
	m_module(_builder.GetInsertBlock()->getParent()->getParent())
{}

}
}
}
