
#pragma once

#include <llvm/IR/Module.h>

#include "Runtime.h"

namespace dev
{
namespace eth
{
namespace jit
{

class ExecutionEngine
{
public:
	// FIXME: constructor? ExecutionEngine();

	int run(std::unique_ptr<llvm::Module> module, RuntimeData* _data, Env* _env);

	bytes returnData;
};

}
}
}
