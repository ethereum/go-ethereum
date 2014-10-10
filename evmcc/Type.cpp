
#include "Type.h"

#include <llvm/IR/DerivedTypes.h>

namespace evmcc
{

llvm::IntegerType* Type::i256;
llvm::PointerType* Type::WordPtr;
llvm::IntegerType* Type::lowPrecision;
llvm::IntegerType* Type::Byte;
llvm::PointerType* Type::BytePtr;
llvm::Type* Type::Void;

void Type::init(llvm::LLVMContext& _context)
{
	i256 = llvm::Type::getIntNTy(_context, 256);
	WordPtr = i256->getPointerTo();
	lowPrecision = llvm::Type::getInt64Ty(_context);
	Byte = llvm::Type::getInt8Ty(_context);
	BytePtr = Byte->getPointerTo();
	Void = llvm::Type::getVoidTy(_context);
}

}
