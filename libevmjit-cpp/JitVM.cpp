
#include "JitVM.h"
#include <libevm/VM.h>
#include <evmjit/libevmjit/ExecutionEngine.h>
#include <evmjit/libevmjit/Utils.h>

namespace dev
{
namespace eth
{

bytesConstRef JitVM::go(ExtVMFace& _ext, OnOpFunc const&, uint64_t)
{
	using namespace jit;

	RuntimeData data;
	data.set(RuntimeData::Gas, m_gas);
	data.set(RuntimeData::Address, fromAddress(_ext.myAddress));
	data.set(RuntimeData::Caller, fromAddress(_ext.caller));
	data.set(RuntimeData::Origin, fromAddress(_ext.origin));
	data.set(RuntimeData::CallValue, _ext.value);
	data.set(RuntimeData::CallDataSize, _ext.data.size());
	data.set(RuntimeData::GasPrice, _ext.gasPrice);
	data.set(RuntimeData::PrevHash, _ext.previousBlock.hash);
	data.set(RuntimeData::CoinBase, fromAddress(_ext.currentBlock.coinbaseAddress));
	data.set(RuntimeData::TimeStamp, _ext.currentBlock.timestamp);
	data.set(RuntimeData::Number, _ext.currentBlock.number);
	data.set(RuntimeData::Difficulty, _ext.currentBlock.difficulty);
	data.set(RuntimeData::GasLimit, _ext.currentBlock.gasLimit);
	data.set(RuntimeData::CodeSize, _ext.code.size());
	data.callData = _ext.data.data();
	data.code = _ext.code.data();

	ExecutionEngine engine;
	auto env = reinterpret_cast<Env*>(&_ext);
	auto exitCode = engine.run(_ext.code, &data, env);

	switch (exitCode)
	{
	case ReturnCode::BadJumpDestination:
		BOOST_THROW_EXCEPTION(BadJumpDestination());
	case ReturnCode::OutOfGas:
		BOOST_THROW_EXCEPTION(OutOfGas());
	case ReturnCode::StackTooSmall:
		BOOST_THROW_EXCEPTION(StackTooSmall());
	case ReturnCode::BadInstruction:
		BOOST_THROW_EXCEPTION(BadInstruction());
	default:
		break;
	}

	m_gas = llvm2eth(data.elems[RuntimeData::Gas]);
	m_output = std::move(engine.returnData);
	return {m_output.data(), m_output.size()};  // TODO: This all bytesConstRef stuff sucks
}

}
}

namespace
{
	// MSVS linker ignores export symbols in Env.cpp if nothing points at least one of them
	extern "C" void env_sload();
	void linkerWorkaround() 
	{ 
		env_sload();
		(void)&linkerWorkaround; // suppress unused function warning from GCC
	}
}
