
#pragma once

#include <llvm/IR/Type.h>

namespace evmcc
{

struct Type
{
	static llvm::Type* i256;

	/// Type for doing low precision arithmetics where 256-bit precision is not supported by native target
	/// @TODO: Use 64-bit for now. In 128-bit compiler-rt library functions are required
	static llvm::Type* lowPrecision;

	static void init(llvm::LLVMContext& _context);
};

}