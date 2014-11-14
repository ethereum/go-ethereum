
#pragma once

#include <llvm/IR/Module.h>

#include <libdevcore/Common.h>
#include <libevm/ExtVMFace.h>

namespace dev
{
namespace eth
{
namespace jit
{

class ExecutionEngine
{
public:
	ExecutionEngine();

	int run(std::unique_ptr<llvm::Module> module, u256& _gas, bool _outputLogs, ExtVMFace* _ext = nullptr);

	bytes returnData;
};

}
}
}
