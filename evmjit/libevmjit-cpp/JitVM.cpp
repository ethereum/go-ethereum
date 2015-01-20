
#include "JitVM.h"
#include <libevm/VM.h>
#include <evmjit/libevmjit/ExecutionEngine.h>
#include "Utils.h"

namespace dev
{
namespace eth
{

bytesConstRef JitVM::go(ExtVMFace& _ext, OnOpFunc const&, uint64_t)
{
	using namespace jit;

	m_data.elems[RuntimeData::Gas]          = eth2llvm(m_gas);
	m_data.elems[RuntimeData::Address]      = eth2llvm(fromAddress(_ext.myAddress));
	m_data.elems[RuntimeData::Caller]       = eth2llvm(fromAddress(_ext.caller));
	m_data.elems[RuntimeData::Origin]       = eth2llvm(fromAddress(_ext.origin));
	m_data.elems[RuntimeData::CallValue]    = eth2llvm(_ext.value);
	m_data.elems[RuntimeData::CallDataSize] = eth2llvm(_ext.data.size());
	m_data.elems[RuntimeData::GasPrice]     = eth2llvm(_ext.gasPrice);
	m_data.elems[RuntimeData::CoinBase]     = eth2llvm(fromAddress(_ext.currentBlock.coinbaseAddress));
	m_data.elems[RuntimeData::TimeStamp]    = eth2llvm(_ext.currentBlock.timestamp);
	m_data.elems[RuntimeData::Number]       = eth2llvm(_ext.currentBlock.number);
	m_data.elems[RuntimeData::Difficulty]   = eth2llvm(_ext.currentBlock.difficulty);
	m_data.elems[RuntimeData::GasLimit]     = eth2llvm(_ext.currentBlock.gasLimit);
	m_data.elems[RuntimeData::CodeSize]     = eth2llvm(_ext.code.size());
	m_data.callData = _ext.data.data();
	m_data.code     = _ext.code.data();

	auto env = reinterpret_cast<Env*>(&_ext);
	auto exitCode = m_engine.run(_ext.code, &m_data, env);
	switch (exitCode)
	{
	case ReturnCode::Suicide:
		_ext.suicide(right160(llvm2eth(m_data.elems[RuntimeData::SuicideDestAddress])));
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
	return {std::get<0>(m_engine.returnData), std::get<1>(m_engine.returnData)};
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
