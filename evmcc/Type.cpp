
#include "Type.h"

#include <llvm/IR/DerivedTypes.h>

namespace dev
{
namespace eth
{
namespace jit
{

llvm::IntegerType* Type::i256;
llvm::PointerType* Type::WordPtr;
llvm::IntegerType* Type::lowPrecision;
llvm::IntegerType* Type::Byte;
llvm::PointerType* Type::BytePtr;
llvm::Type* Type::Void;
llvm::IntegerType* Type::MainReturn;

void Type::init(llvm::LLVMContext& _context)
{
	i256 = llvm::Type::getIntNTy(_context, 256);
	WordPtr = i256->getPointerTo();
	lowPrecision = llvm::Type::getInt64Ty(_context);
	Byte = llvm::Type::getInt8Ty(_context);
	BytePtr = Byte->getPointerTo();
	Void = llvm::Type::getVoidTy(_context);
	MainReturn = llvm::Type::getInt32Ty(_context);
}

llvm::ConstantInt* Constant::get(uint64_t _n)
{
	return llvm::ConstantInt::get(Type::i256, _n);
}

llvm::ConstantInt* Constant::get(ReturnCode _returnCode)
{
	return llvm::ConstantInt::get(Type::MainReturn, static_cast<uint64_t>(_returnCode));
}

}
}
}

