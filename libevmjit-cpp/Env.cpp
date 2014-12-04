
#include <llvm/IR/Function.h>
#include <llvm/IR/TypeBuilder.h>
#include <llvm/IR/IntrinsicInst.h>

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

	EXPORT void ext_store(ExtVMFace* _env, i256* _index, i256* o_value)
	{
		auto index = llvm2eth(*_index);
		auto value = _env->store(index); // Interface uses native endianness
		*o_value = eth2llvm(value);
	}

	EXPORT void ext_setStore(ExtVMFace* _env, i256* _index, i256* _value)
	{
		auto index = llvm2eth(*_index);
		auto value = llvm2eth(*_value);

		if (value == 0 && _env->store(index) != 0)	// If delete
			_env->sub.refunds += c_sstoreRefundGas;	// Increase refund counter

		_env->setStore(index, value);	// Interface uses native endianness
	}

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

	EXPORT void ext_create(ExtVMFace* _env, i256* _endowment, i256* _initOff, i256* _initSize, h256* o_address)
	{
		auto endowment = llvm2eth(*_endowment);

		if (_env->balance(_env->myAddress) >= endowment)
		{
			_env->subBalance(endowment);
			u256 gas;   // TODO: Handle gas
			auto initOff = static_cast<size_t>(llvm2eth(*_initOff));
			auto initSize = static_cast<size_t>(llvm2eth(*_initSize));
			//auto&& initRef = bytesConstRef(_rt->getMemory().data() + initOff, initSize);
			auto initRef = bytesConstRef(); // FIXME: Handle memory
			OnOpFunc onOp {}; // TODO: Handle that thing
			h256 address(_env->create(endowment, &gas, initRef, onOp), h256::AlignRight);
			*o_address = address;
		}
		else
			*o_address = {};
	}

	EXPORT void ext_call(ExtVMFace* _env, i256* io_gas, h256* _receiveAddress, i256* _value, i256* _inOff, i256* _inSize, i256* _outOff, i256* _outSize, h256* _codeAddress, i256* o_ret)
	{
		auto&& ext = *_env;
		auto value = llvm2eth(*_value);

		auto ret = false;
		auto gas = llvm2eth(*io_gas);
		if (ext.balance(ext.myAddress) >= value)
		{
			ext.subBalance(value);
			auto receiveAddress = right160(*_receiveAddress);
			auto inOff = static_cast<size_t>(llvm2eth(*_inOff));
			auto inSize = static_cast<size_t>(llvm2eth(*_inSize));
			auto outOff = static_cast<size_t>(llvm2eth(*_outOff));
			auto outSize = static_cast<size_t>(llvm2eth(*_outSize));
			//auto&& inRef = bytesConstRef(_rt->getMemory().data() + inOff, inSize);
			//auto&& outRef = bytesConstRef(_rt->getMemory().data() + outOff, outSize);
			auto inRef = bytesConstRef(); // FIXME: Handle memory
			auto outRef = bytesConstRef(); // FIXME: Handle memory
			OnOpFunc onOp {}; // TODO: Handle that thing
			auto codeAddress = right160(*_codeAddress);
			ret = ext.call(receiveAddress, value, inRef, &gas, outRef, onOp, {}, codeAddress);
		}

		*io_gas = eth2llvm(gas);
		o_ret->a = ret ? 1 : 0;
	}

	EXPORT void ext_sha3(ExtVMFace* _env, i256* _inOff, i256* _inSize, i256* o_ret)
	{
		auto inOff = static_cast<size_t>(llvm2eth(*_inOff));
		auto inSize = static_cast<size_t>(llvm2eth(*_inSize));
		//auto dataRef = bytesConstRef(_rt->getMemory().data() + inOff, inSize);
		auto dataRef = bytesConstRef(); // FIXME: Handle memory
		auto hash = sha3(dataRef);
		*o_ret = *reinterpret_cast<i256*>(&hash);
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

	void ext_show_bytes(bytesConstRef _bytes)
	{
		for (auto b : _bytes)
			std::cerr << std::hex << std::setw(2) << std::setfill('0') << static_cast<unsigned>(b) << " ";
		std::cerr << std::endl;
	}

	EXPORT void ext_log0(ExtVMFace* _env, i256* _memIdx, i256* _numBytes)
	{
		auto memIdx = llvm2eth(*_memIdx).convert_to<size_t>();
		auto numBytes = llvm2eth(*_numBytes).convert_to<size_t>();

		//auto dataRef = bytesConstRef(_rt->getMemory().data() + memIdx, numBytes);
		auto dataRef = bytesConstRef(); // FIXME: Handle memory
		_env->log({}, dataRef);
	}

	EXPORT void ext_log1(ExtVMFace* _env, i256* _memIdx, i256* _numBytes, i256* _topic1)
	{
		auto memIdx = static_cast<size_t>(llvm2eth(*_memIdx));
		auto numBytes = static_cast<size_t>(llvm2eth(*_numBytes));

		auto topic1 = llvm2eth(*_topic1);

		//auto dataRef = bytesConstRef(_rt->getMemory().data() + memIdx, numBytes);
		auto dataRef = bytesConstRef(); // FIXME: Handle memory
		_env->log({topic1}, dataRef);
	}

	EXPORT void ext_log2(ExtVMFace* _env, i256* _memIdx, i256* _numBytes, i256* _topic1, i256* _topic2)
	{
		auto memIdx = static_cast<size_t>(llvm2eth(*_memIdx));
		auto numBytes = static_cast<size_t>(llvm2eth(*_numBytes));

		auto topic1 = llvm2eth(*_topic1);
		auto topic2 = llvm2eth(*_topic2);

		//auto dataRef = bytesConstRef(_rt->getMemory().data() + memIdx, numBytes);
		auto dataRef = bytesConstRef(); // FIXME: Handle memory
		_env->log({ topic1, topic2 }, dataRef);
	}

	EXPORT void ext_log3(ExtVMFace* _env, i256* _memIdx, i256* _numBytes, i256* _topic1, i256* _topic2, i256* _topic3)
	{
		auto memIdx = static_cast<size_t>(llvm2eth(*_memIdx));
		auto numBytes = static_cast<size_t>(llvm2eth(*_numBytes));

		auto topic1 = llvm2eth(*_topic1);
		auto topic2 = llvm2eth(*_topic2);
		auto topic3 = llvm2eth(*_topic3);

		//auto dataRef = bytesConstRef(_rt->getMemory().data() + memIdx, numBytes);
		auto dataRef = bytesConstRef(); // FIXME: Handle memory
		_env->log({ topic1, topic2, topic3 }, dataRef);
	}

	EXPORT void ext_log4(ExtVMFace* _env, i256* _memIdx, i256* _numBytes, i256* _topic1, i256* _topic2, i256* _topic3, i256* _topic4)
	{
		auto memIdx = static_cast<size_t>(llvm2eth(*_memIdx));
		auto numBytes = static_cast<size_t>(llvm2eth(*_numBytes));

		auto topic1 = llvm2eth(*_topic1);
		auto topic2 = llvm2eth(*_topic2);
		auto topic3 = llvm2eth(*_topic3);
		auto topic4 = llvm2eth(*_topic4);

		//auto dataRef = bytesConstRef(_rt->getMemory().data() + memIdx, numBytes);
		auto dataRef = bytesConstRef(); // FIXME: Handle memory
		_env->log({ topic1, topic2, topic3, topic4 }, dataRef);
	}
}

