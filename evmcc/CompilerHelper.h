
#pragma once

#include <llvm/IR/IRBuilder.h>


namespace dev
{
namespace eth
{
namespace jit
{

class CompilerHelper
{
protected:
	CompilerHelper(llvm::IRBuilder<>& _builder);

	CompilerHelper(const CompilerHelper&) = delete;
	void operator=(CompilerHelper) = delete;

	/// Reference to parent compiler IR builder
	llvm::IRBuilder<>& m_builder;

	/// Reference to the IR module being compiled
	llvm::Module* m_module;
};

}
}
}
