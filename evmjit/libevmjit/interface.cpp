#include "interface.h"
#include <cstdio>
#include "ExecutionEngine.h"

extern "C"
{

int evmjit_run()
{
	using namespace dev::eth::jit;

	ExecutionEngine engine;
	u256 gas = 100000;

	bytes bytecode = { 0x60, 0x01 };

	// Create random runtime data
	RuntimeData data;
	data.set(RuntimeData::Gas, gas);
	data.set(RuntimeData::Address, 0);
	data.set(RuntimeData::Caller, 0);
	data.set(RuntimeData::Origin, 0);
	data.set(RuntimeData::CallValue, 0xabcd);
	data.set(RuntimeData::CallDataSize, 3);
	data.set(RuntimeData::GasPrice, 1003);
	data.set(RuntimeData::CoinBase, 0);
	data.set(RuntimeData::TimeStamp, 1005);
	data.set(RuntimeData::Number, 1006);
	data.set(RuntimeData::Difficulty, 16);
	data.set(RuntimeData::GasLimit, 1008);
	data.set(RuntimeData::CodeSize, bytecode.size());
	data.callData = (uint8_t*)"abc";
	data.code = bytecode.data();

	// BROKEN: env_* functions must be implemented & RuntimeData struct created
	// TODO: Do not compile module again
	auto result = engine.run(bytecode, &data, nullptr);
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
