
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

	m_data.set(RuntimeData::Gas, m_gas);
	m_data.set(RuntimeData::Address, fromAddress(_ext.myAddress));
	m_data.set(RuntimeData::Caller, fromAddress(_ext.caller));
	m_data.set(RuntimeData::Origin, fromAddress(_ext.origin));
	m_data.set(RuntimeData::CallValue, _ext.value);
	m_data.set(RuntimeData::CallDataSize, _ext.data.size());
	m_data.set(RuntimeData::GasPrice, _ext.gasPrice);
	m_data.set(RuntimeData::PrevHash, _ext.previousBlock.hash);
	m_data.set(RuntimeData::CoinBase, fromAddress(_ext.currentBlock.coinbaseAddress));
	m_data.set(RuntimeData::TimeStamp, _ext.currentBlock.timestamp);
	m_data.set(RuntimeData::Number, _ext.currentBlock.number);
	m_data.set(RuntimeData::Difficulty, _ext.currentBlock.difficulty);
	m_data.set(RuntimeData::GasLimit, _ext.currentBlock.gasLimit);
	m_data.set(RuntimeData::CodeSize, _ext.code.size());
	m_data.callData = _ext.data.data();
	m_data.code = _ext.code.data();

	auto env = reinterpret_cast<Env*>(&_ext);
	auto exitCode = m_engine.run(_ext.code, &m_data, env);
	switch (exitCode)
	{
	case ReturnCode::Suicide:
		_ext.suicide(right160(m_data.get(RuntimeData::SuicideDestAddress)));
		break;

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

	m_gas = llvm2eth(m_data.elems[RuntimeData::Gas]);
	return {m_engine.returnData.data(), m_engine.returnData.size()};  // TODO: This all bytesConstRef is problematic, review.
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
