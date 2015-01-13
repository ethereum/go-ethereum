
#include "Type.h"

#include <llvm/IR/DerivedTypes.h>

#include "RuntimeManager.h"

namespace dev
{
namespace eth
{
namespace jit
{

llvm::IntegerType* Type::Word;
llvm::PointerType* Type::WordPtr;
llvm::IntegerType* Type::lowPrecision;
llvm::IntegerType* Type::Bool;
llvm::IntegerType* Type::Size;
llvm::IntegerType* Type::Byte;
llvm::PointerType* Type::BytePtr;
llvm::Type* Type::Void;
llvm::IntegerType* Type::MainReturn;
llvm::PointerType* Type::EnvPtr;
llvm::PointerType* Type::RuntimeDataPtr;
llvm::PointerType* Type::RuntimePtr;

void Type::init(llvm::LLVMContext& _context)
{
	if (!Word)	// Do init only once
	{
		Word = llvm::Type::getIntNTy(_context, 256);
		WordPtr = Word->getPointerTo();
		lowPrecision = llvm::Type::getInt64Ty(_context);
		// TODO: Size should be architecture-dependent
		Bool = llvm::Type::getInt1Ty(_context);
		Size = llvm::Type::getInt64Ty(_context);
		Byte = llvm::Type::getInt8Ty(_context);
		BytePtr = Byte->getPointerTo();
		Void = llvm::Type::getVoidTy(_context);
		MainReturn = llvm::Type::getInt32Ty(_context);

		EnvPtr = llvm::StructType::create(_context, "Env")->getPointerTo();
		RuntimeDataPtr = RuntimeManager::getRuntimeDataType()->getPointerTo();
		RuntimePtr = RuntimeManager::getRuntimeType()->getPointerTo();
	}
}

llvm::ConstantInt* Constant::get(int64_t _n)
{
	return llvm::ConstantInt::getSigned(Type::Word, _n);
}

llvm::ConstantInt* Constant::get(llvm::APInt const& _n)
{
	return llvm::ConstantInt::get(Type::Word->getContext(), _n);
}

llvm::ConstantInt* Constant::get(ReturnCode _returnCode)
{
	return llvm::ConstantInt::get(Type::MainReturn, static_cast<uint64_t>(_returnCode));
}

}
}
}

