
#pragma once

#include <llvm/IR/Type.h>
#include <llvm/IR/Constants.h>
#include "Common.h"

namespace dev
{
namespace eth
{
namespace jit
{

struct Type
{
	static llvm::IntegerType* Word;
	static llvm::PointerType* WordPtr;

	/// Type for doing low precision arithmetics where 256-bit precision is not supported by native target
	/// @TODO: Use 64-bit for now. In 128-bit compiler-rt library functions are required
	static llvm::IntegerType* lowPrecision;

	static llvm::IntegerType* Bool;
	static llvm::IntegerType* Size;

	static llvm::IntegerType* Byte;
	static llvm::PointerType* BytePtr;

	static llvm::Type* Void;

	/// Main function return type
	static llvm::IntegerType* MainReturn;

	static llvm::PointerType* EnvPtr;
	static llvm::PointerType* RuntimeDataPtr;
	static llvm::PointerType* RuntimePtr;

	static void init(llvm::LLVMContext& _context);
};

struct Constant
{
	/// Returns word-size constant
	static llvm::ConstantInt* get(int64_t _n);
	static llvm::ConstantInt* get(llvm::APInt const& _n);

	static llvm::ConstantInt* get(ReturnCode _returnCode);
};

}
}
}

