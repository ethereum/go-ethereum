
#pragma once

#include <llvm/IR/Module.h>

#include <libdevcore/Common.h>

namespace evmcc
{

class Compiler
{
public:

	Compiler();

	std::unique_ptr<llvm::Module> compile(const dev::bytes& bytecode);

};

}
