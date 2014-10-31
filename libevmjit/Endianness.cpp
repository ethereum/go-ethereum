
#include "Endianness.h"

#include <llvm/IR/IntrinsicInst.h>

#include "Type.h"

namespace dev
{
namespace eth
{
namespace jit
{

llvm::Value* Endianness::bswap(llvm::IRBuilder<>& _builder, llvm::Value* _word)
{
	// TODO: Native is Little Endian
	assert(_word->getType() == Type::Word);
	auto bswap = llvm::Intrinsic::getDeclaration(_builder.GetInsertBlock()->getParent()->getParent(), llvm::Intrinsic::bswap, Type::Word);
	return _builder.CreateCall(bswap, _word);
}

}
}
}
