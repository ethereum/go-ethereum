
#include "VM.h"

#include <libevm/VMFace.h>

#include "ExecutionEngine.h"
#include "Compiler.h"
#include "Type.h"

namespace dev
{
namespace eth
{
namespace jit
{

bytesConstRef VM::go(ExtVMFace& _ext, OnOpFunc const&, uint64_t)
{
	auto module = Compiler().compile(_ext.code);

	ExecutionEngine engine;
	auto exitCode = engine.run(std::move(module), m_gas, &_ext);

	switch (static_cast<ReturnCode>(exitCode))
	{
	case ReturnCode::BadJumpDestination:
		BOOST_THROW_EXCEPTION(BadJumpDestination());
	case ReturnCode::OutOfGas:
		BOOST_THROW_EXCEPTION(OutOfGas());
	case ReturnCode::StackTooSmall:
		BOOST_THROW_EXCEPTION(StackTooSmall(1, 0));
	case ReturnCode::BadInstruction:
		BOOST_THROW_EXCEPTION(BadInstruction());
	default:
		break;
	}

	m_output = std::move(engine.returnData);
	return {m_output.data(), m_output.size()};  // TODO: This all bytesConstRef stuff sucks
}

}
}
}
