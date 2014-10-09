
#pragma once

#include <llvm/IR/Type.h>

namespace evmcc
{

struct Type
{
	static llvm::Type* i256;

	static void init(llvm::LLVMContext& _context);
};

}