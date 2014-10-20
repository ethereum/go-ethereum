
#include "Endianness.h"

#include <llvm/IR/IntrinsicInst.h>

#include "Type.h"

namespace dev
{
namespace eth
{
namespace jit
{

llvm::Value* Endianness::toBE(llvm::IRBuilder<>& _builder, llvm::Value* _word)
{
	// TODO: Native is Little Endian
	auto bswap = llvm::Intrinsic::getDeclaration(_builder.GetInsertBlock()->getParent()->getParent(), llvm::Intrinsic::bswap, Type::i256);
	return _builder.CreateCall(bswap, _word);
}

}
}
}
