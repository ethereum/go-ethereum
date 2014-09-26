
#pragma once

#include <llvm/IR/Module.h>

#include <libdevcore/Common.h>

namespace evmcc
{

class ExecutionEngine
{
public:
	ExecutionEngine();

	int run(std::unique_ptr<llvm::Module> module);
};

}