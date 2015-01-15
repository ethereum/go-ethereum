#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct evmjit_result
{
	int32_t  returnCode;
	uint64_t returnDataSize;
	void*    returnData;

} evmjit_result;

evmjit_result evmjit_run(void* _data, void* _env);

// JIT object opaque type
typedef struct evm_jit evm_jit;

// Contract execution return code
typedef int evm_jit_return_code;

// Host-endian 256-bit integer type
typedef struct i256 i256;

struct i256
{
	char b[33];
};

// Big-endian right aligned 256-bit hash
typedef struct h256 h256;

// Runtime data struct - must be provided by external language (Go, C++, Python)
typedef struct evm_jit_rt evm_jit_rt;

// Runtime callback functions  - implementations must be provided by external language (Go, C++, Python)
void evm_jit_rt_sload(evm_jit_rt* _rt, i256* _index, i256* _ret);
void evm_jit_rt_sstore(evm_jit_rt* _rt, i256* _index, i256* _value);
void evm_jit_rt_balance(evm_jit_rt* _rt, h256* _address, i256* _ret);
// And so on...

evm_jit* evm_jit_create(evm_jit_rt* _runtime_data);

evm_jit_return_code evm_jit_execute(evm_jit* _jit);

void evm_jit_get_return_data(evm_jit* _jit, char* _return_data_offset, size_t* _return_data_size);

void evm_jit_destroy(evm_jit* _jit);

#ifdef __cplusplus
}
#endif
