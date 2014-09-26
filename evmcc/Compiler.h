
#pragma once

#include <llvm/IR/Type.h>

#include <libdevcore/Common.h>

namespace evmcc
{

class Compiler
{

private:

	struct
	{
		llvm::Type* word8;
		llvm::Type* word8ptr;
		llvm::Type* word256;
		llvm::Type* word256ptr;
		llvm::Type* word256arr;
		llvm::Type* size;
	} Types;

public:

	Compiler();

	void compile(const dev::bytes& bytecode);

};

}