#pragma once

namespace llvm
{
	class Module;
}

#include "RuntimeData.h"

namespace dev
{
namespace eth
{
namespace jit
{
class ExecBundle;

class ExecutionEngine
{
public:
	ExecutionEngine() = default;
	ExecutionEngine(ExecutionEngine const&) = delete;
	void operator=(ExecutionEngine) = delete;

	ReturnCode run(bytes const& _code, RuntimeData* _data, Env* _env);
	ReturnCode run(std::unique_ptr<llvm::Module> module, RuntimeData* _data, Env* _env, bytes const& _code);

	bytes returnData;

private:
	ReturnCode run(ExecBundle const& _exec, RuntimeData* _data, Env* _env);
};

}
}
}
