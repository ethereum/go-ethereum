
#include "Type.h"

#include <llvm/IR/DerivedTypes.h>

#include "Runtime.h"

namespace dev
{
namespace eth
{
namespace jit
{

llvm::IntegerType* Type::Word;
llvm::PointerType* Type::WordPtr;
llvm::IntegerType* Type::lowPrecision;
llvm::IntegerType* Type::Size;
llvm::IntegerType* Type::Byte;
llvm::PointerType* Type::BytePtr;
llvm::Type* Type::Void;
llvm::IntegerType* Type::MainReturn;
llvm::PointerType* Type::RuntimePtr;

void Type::init(llvm::LLVMContext& _context)
{
	Word = llvm::Type::getIntNTy(_context, 256);
	WordPtr = Word->getPointerTo();
	lowPrecision = llvm::Type::getInt64Ty(_context);
	// TODO: Size should be architecture-dependent
	Size = llvm::Type::getInt64Ty(_context);
	Byte = llvm::Type::getInt8Ty(_context);
	BytePtr = Byte->getPointerTo();
	Void = llvm::Type::getVoidTy(_context);
	MainReturn = llvm::Type::getInt32Ty(_context);
	RuntimePtr = RuntimeData::getType()->getPointerTo();
}

llvm::ConstantInt* Constant::get(int64_t _n)
{
	return llvm::ConstantInt::getSigned(Type::Word, _n);
}

llvm::ConstantInt* Constant::get(u256 _n)
{
	llvm::APInt n(256, _n.str(0, std::ios_base::hex), 16);
	assert(n.toString(10, false) == _n.str());
	return static_cast<llvm::ConstantInt*>(llvm::ConstantInt::get(Type::Word, n));
}

llvm::ConstantInt* Constant::get(ReturnCode _returnCode)
{
	return llvm::ConstantInt::get(Type::MainReturn, static_cast<uint64_t>(_returnCode));
}

}
}
}

