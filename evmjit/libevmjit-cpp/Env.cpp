
#include <libdevcrypto/SHA3.h>
#include <libevm/FeeStructure.h>
#include <libevm/ExtVMFace.h>

#include "Utils.h"

extern "C"
{
	#ifdef _MSC_VER
		#define EXPORT __declspec(dllexport)
	#else
		#define EXPORT
	#endif

	using namespace dev;
	using namespace dev::eth;
	using jit::i256;

	EXPORT void env_sload(ExtVMFace* _env, i256* _index, i256* o_value)
	{
		auto index = llvm2eth(*_index);
		auto value = _env->store(index); // Interface uses native endianness
		*o_value = eth2llvm(value);
	}

	EXPORT void env_sstore(ExtVMFace* _env, i256* _index, i256* _value)
	{
		auto index = llvm2eth(*_index);
		auto value = llvm2eth(*_value);

		if (value == 0 && _env->store(index) != 0)	// If delete
			_env->sub.refunds += c_sstoreRefundGas;	// Increase refund counter

		_env->setStore(index, value);	// Interface uses native endianness
	}

	EXPORT void env_balance(ExtVMFace* _env, h256* _address, i256* o_value)
	{
		auto u = _env->balance(right160(*_address));
		*o_value = eth2llvm(u);
	}

	EXPORT void env_blockhash(ExtVMFace* _env, i256* _number, h256* o_hash)
	{
		*o_hash = _env->blockhash(llvm2eth(*_number));
	}

	EXPORT void env_create(ExtVMFace* _env, i256* io_gas, i256* _endowment, byte* _initBeg, uint64_t _initSize, h256* o_address)
	{
		auto endowment = llvm2eth(*_endowment);
		if (_env->balance(_env->myAddress) >= endowment && _env->depth < 1024)
		{
			_env->subBalance(endowment);
			auto gas = llvm2eth(*io_gas);
			OnOpFunc onOp {}; // TODO: Handle that thing
			h256 address(_env->create(endowment, gas, {_initBeg, _initSize}, onOp), h256::AlignRight);
			*io_gas = eth2llvm(gas);
			*o_address = address;
		}
		else
			*o_address = {};
	}

	EXPORT bool env_call(ExtVMFace* _env, i256* io_gas, h256* _receiveAddress, i256* _value, byte* _inBeg, uint64_t _inSize, byte* _outBeg, uint64_t _outSize, h256* _codeAddress)
	{
		auto value = llvm2eth(*_value);
		if (_env->balance(_env->myAddress) >= value && _env->depth < 1024)
		{
			_env->subBalance(value);
			auto receiveAddress = right160(*_receiveAddress);
			auto inRef = bytesConstRef{_inBeg, _inSize};
			auto outRef = bytesConstRef{_outBeg, _outSize};
			OnOpFunc onOp {}; // TODO: Handle that thing
			auto codeAddress = right160(*_codeAddress);
			auto gas = llvm2eth(*io_gas);
			auto ret = _env->call(receiveAddress, value, inRef, gas, outRef, onOp, {}, codeAddress);
			*io_gas = eth2llvm(gas);
			return ret;
		}

		return false;
	}

	EXPORT void env_sha3(byte* _begin, uint64_t _size, h256* o_hash)
	{
		auto hash = sha3({_begin, _size});
		*o_hash = hash;
	}

	EXPORT byte const* env_extcode(ExtVMFace* _env, h256* _addr256, uint64_t* o_size)
	{
		auto addr = right160(*_addr256);
		auto& code = _env->codeAt(addr);
		*o_size = code.size();
		return code.data();
	}

	EXPORT void env_log(ExtVMFace* _env, byte* _beg, uint64_t _size, h256* _topic1, h256* _topic2, h256* _topic3, h256* _topic4)
	{
		dev::h256s topics;

		if (_topic1)
			topics.push_back(*_topic1);

		if (_topic2)
			topics.push_back(*_topic2);

		if (_topic3)
			topics.push_back(*_topic3);

		if (_topic4)
			topics.push_back(*_topic4);

		_env->log(std::move(topics), {_beg, _size});
	}
}

