
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

	static llvm::Value* toBE(llvm::IRBuilder<>& _builder, llvm::Value* _word);
};

}
}
}
