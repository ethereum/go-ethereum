
#include "VM.h"

#include <libevm/VMFace.h>
#include <libevm/VM.h>

#include "../libevmjit/ExecutionEngine.h"
#include "../libevmjit/Compiler.h"

namespace dev
{
namespace eth
{
namespace jit
{

bytesConstRef VM::go(ExtVMFace& _ext, OnOpFunc const&, uint64_t)
{
	Compiler::Options defaultOptions;
	auto module = Compiler(defaultOptions).compile(_ext.code);

	RuntimeData data = {};

#define set(INDEX, VALUE) data.elems[INDEX] = eth2llvm(VALUE)
	set(RuntimeData::Gas, m_gas);
	set(RuntimeData::Address, fromAddress(_ext.myAddress));
	set(RuntimeData::Caller, fromAddress(_ext.caller));
	set(RuntimeData::Origin, fromAddress(_ext.origin));
	set(RuntimeData::CallValue, _ext.value);
	set(RuntimeData::CallDataSize, _ext.data.size());
	set(RuntimeData::GasPrice, _ext.gasPrice);
	set(RuntimeData::PrevHash, _ext.previousBlock.hash);
	set(RuntimeData::CoinBase, fromAddress(_ext.currentBlock.coinbaseAddress));
	set(RuntimeData::TimeStamp, _ext.currentBlock.timestamp);
	set(RuntimeData::Number, _ext.currentBlock.number);
	set(RuntimeData::Difficulty, _ext.currentBlock.difficulty);
	set(RuntimeData::GasLimit, _ext.currentBlock.gasLimit);
	set(RuntimeData::CodeSize, _ext.code.size());   // TODO: Use constant
	data.callData = _ext.data.data();
	data.code = _ext.code.data();
#undef set

	ExecutionEngine engine;
	auto env = reinterpret_cast<Env*>(&_ext);
	auto exitCode = engine.run(std::move(module), &data, env);

	switch (static_cast<ReturnCode>(exitCode))
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
}

namespace
{
	// MSVS linker ignores export symbols in Env.cpp if nothing point at least one of them
	extern "C" void ext_store();
	void linkerWorkaround() { ext_store(); }
}
