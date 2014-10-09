
#include "Type.h"

#include <llvm/IR/DerivedTypes.h>

namespace evmcc
{

llvm::Type* Type::i256;

void Type::init(llvm::LLVMContext& _context)
{
	i256 = llvm::Type::getIntNTy(_context, 256);
}

}