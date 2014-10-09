
#include "Type.h"

#include <llvm/IR/DerivedTypes.h>

namespace evmcc
{

llvm::Type* Type::i256;
llvm::Type* Type::lowPrecision;
llvm::Type* Type::Void;

void Type::init(llvm::LLVMContext& _context)
{
	i256 = llvm::Type::getIntNTy(_context, 256);
	lowPrecision = llvm::Type::getInt64Ty(_context);
	Void = llvm::Type::getVoidTy(_context);
}

}