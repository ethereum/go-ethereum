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

	/// Reference to returned data (RETURN opcode used)
	bytes_ref returnData;

private:
	/// After execution, if RETURN used, memory is moved there
	/// to allow client copy the returned data
	bytes m_memory;
};

}
}
}
