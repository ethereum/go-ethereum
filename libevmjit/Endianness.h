
#pragma once

#include <boost/detail/endian.hpp>

#include <llvm/IR/IRBuilder.h>

namespace dev
{
namespace eth
{
namespace jit
{

struct Endianness
{

#if defined(BOOST_LITTLE_ENDIAN)
	static llvm::Value* toBE(llvm::IRBuilder<>& _builder, llvm::Value* _word) { return bswap(_builder, _word); }
	static llvm::Value* toNative(llvm::IRBuilder<>& _builder, llvm::Value* _word) { return bswap(_builder, _word); }

#elif defined(BOOST_BIG_ENDIAN)
	static llvm::Value* toBE(llvm::IRBuilder<>&, llvm::Value* _word) { return _word; }
	static llvm::Value* toNative(llvm::IRBuilder<>&, llvm::Value* _word) { return _word; }

#endif	// Add support for PDP endianness if needed

private:
	static llvm::Value* bswap(llvm::IRBuilder<>& _builder, llvm::Value* _word);
};

}
}
}
