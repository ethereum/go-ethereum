#include "interface.h"
#include <cstdio>
#include "ExecutionEngine.h"

extern "C"
{

int evmjit_run(void* _data, void* _env)
{
	using namespace dev::eth::jit;

	auto data = static_cast<RuntimeData*>(_data);

	std::cerr << "GAS: " << data->elems[RuntimeData::Gas].a << "\n";

	ExecutionEngine engine;

	auto codePtr = data->code;
	auto codeSize = data->elems[RuntimeData::CodeSize].a;
	bytes bytecode;
	bytecode.insert(bytecode.end(), codePtr, codePtr + codeSize);

	auto result = engine.run(bytecode, data, static_cast<Env*>(_env));
	return static_cast<int>(result);
}

// Runtime callback functions  - implementations must be provided by external language (Go, C++, Python)
void evm_jit_rt_sload(evm_jit_rt* _rt, i256* _index, i256* _ret);
void evm_jit_rt_sstore(evm_jit_rt* _rt, i256* _index, i256* _value);
void evm_jit_rt_balance(evm_jit_rt* _rt, h256* _address, i256* _ret);
// And so on...

evm_jit* evm_jit_create(evm_jit_rt*)
{
	printf("EVM JIT create");

	int* a = nullptr;
	*a = 1;

	return nullptr;
}

evm_jit_return_code evm_jit_execute(evm_jit* _jit);

void evm_jit_get_return_data(evm_jit* _jit, char* _return_data_offset, size_t* _return_data_size);

void evm_jit_destroy(evm_jit* _jit);

}
