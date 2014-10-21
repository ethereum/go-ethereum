#pragma once

#include "CompilerHelper.h"

#include <llvm/IR/Module.h>

namespace dev
{
namespace eth
{
namespace jit
{

class Stack : public CompilerHelper
{
public:
	Stack(llvm::IRBuilder<>& builder);
	virtual ~Stack();

	void pushWord(llvm::Value* _word);
	llvm::Instruction* popWord();

private:
	llvm::Function* m_push;
	llvm::Function* m_pop;

	llvm::Value* m_arg;
};


}
}
}


