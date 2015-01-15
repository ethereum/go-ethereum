#include "interface.h"
#include <cstdio>
#include "ExecutionEngine.h"

extern "C"
{

evmjit_result evmjit_run(void* _data, void* _env)
{
	using namespace dev::eth::jit;

	auto data = static_cast<RuntimeData*>(_data);

	ExecutionEngine engine;

	auto codePtr = data->code;
	auto codeSize = data->elems[RuntimeData::CodeSize].a;
	bytes bytecode;
	bytecode.insert(bytecode.end(), codePtr, codePtr + codeSize);

	auto returnCode = engine.run(bytecode, data, static_cast<Env*>(_env));
	evmjit_result result = {static_cast<int32_t>(returnCode), 0, nullptr};
	if (returnCode == ReturnCode::Return && !engine.returnData.empty())
	{
		// TODO: Optimized returning data. Allocating memory on client side by callback function might be a good idea
		result.returnDataSize = engine.returnData.size();
		result.returnData = std::malloc(result.returnDataSize);
		std::memcpy(result.returnData, engine.returnData.data(), result.returnDataSize);
	}

	return result;
}

}
