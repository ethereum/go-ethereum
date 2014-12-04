
#include <libdevcrypto/SHA3.h>
#include <libevm/FeeStructure.h>
#include <libevm/ExtVMFace.h>

#include "../libevmjit/Utils.h"

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
	using jit::eth2llvm;

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

	// TODO: Move to Memory/Runtime
	EXPORT void ext_calldataload(ExtVMFace* _env, i256* _index, ::byte* o_value)
	{
		auto index = static_cast<size_t>(llvm2eth(*_index));
		assert(index + 31 > index); // TODO: Handle large index
		for (size_t i = index, j = 0; i <= index + 31; ++i, ++j)
			o_value[j] = i < _env->data.size() ? _env->data[i] : 0; // Keep Big Endian
		// TODO: It all can be done by adding padding to data or by using min() algorithm without branch
	}

	EXPORT void ext_balance(ExtVMFace* _env, h256* _address, i256* o_value)
	{
		auto u = _env->balance(right160(*_address));
		*o_value = eth2llvm(u);
	}

	EXPORT void ext_suicide(ExtVMFace* _env, h256 const* _address)
	{
		_env->suicide(right160(*_address));
	}

	EXPORT void env_create(ExtVMFace* _env, i256* _endowment, byte* _initBeg, uint64_t _initSize, h256* o_address)
	{
		auto endowment = llvm2eth(*_endowment);

		if (_env->balance(_env->myAddress) >= endowment)
		{
			_env->subBalance(endowment);
			u256 gas;   // TODO: Handle gas
			OnOpFunc onOp {}; // TODO: Handle that thing
			h256 address(_env->create(endowment, &gas, {_initBeg, _initSize}, onOp), h256::AlignRight);
			*o_address = address;
		}
		else
			*o_address = {};
	}

	EXPORT bool env_call(ExtVMFace* _env, i256* io_gas, h256* _receiveAddress, i256* _value, byte* _inBeg, uint64_t _inSize, byte* _outBeg, uint64_t _outSize, h256* _codeAddress)
	{
		auto value = llvm2eth(*_value);
		if (_env->balance(_env->myAddress) >= value)
		{
			_env->subBalance(value);
			auto receiveAddress = right160(*_receiveAddress);
			auto inRef = bytesConstRef{_inBeg, _inSize};
			auto outRef = bytesConstRef{_outBeg, _outSize};
			OnOpFunc onOp {}; // TODO: Handle that thing
			auto codeAddress = right160(*_codeAddress);
			auto gas = llvm2eth(*io_gas);
			auto ret = _env->call(receiveAddress, value, inRef, &gas, outRef, onOp, {}, codeAddress);
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

	EXPORT unsigned char* ext_codeAt(ExtVMFace* _env, h256* _addr256)
	{
		auto addr = right160(*_addr256);
		auto& code = _env->codeAt(addr);
		return const_cast<unsigned char*>(code.data());
	}

	EXPORT void ext_codesizeAt(ExtVMFace* _env, h256* _addr256, i256* o_ret)
	{
		auto addr = right160(*_addr256);
		auto& code = _env->codeAt(addr);
		*o_ret = eth2llvm(u256(code.size()));
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

