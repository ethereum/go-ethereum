
#pragma once

#include <llvm/IR/IRBuilder.h>

namespace dev
{
namespace eth
{
namespace jit
{

class Endianness
{
public:

	static llvm::Value* toBE(llvm::IRBuilder<>& _builder, llvm::Value* _word) { return bswap(_builder, _word); }

	static llvm::Value* toNative(llvm::IRBuilder<>& _builder, llvm::Value* _word) { return bswap(_builder, _word); }

private:
	static llvm::Value* bswap(llvm::IRBuilder<>& _builder, llvm::Value* _word);
};

}
}
}
