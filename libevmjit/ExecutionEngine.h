#pragma once

#include "RuntimeData.h"

namespace dev
{
namespace eth
{
namespace jit
{

class ExecutionEngine
{
public:
	ExecutionEngine() = default;
	ExecutionEngine(ExecutionEngine const&) = delete;
	void operator=(ExecutionEngine) = delete;

	ReturnCode run(bytes const& _code, RuntimeData* _data, Env* _env);

	bytes returnData;
};

}
}
}
