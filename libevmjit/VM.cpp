
#include "VM.h"

#include <libevm/VMFace.h>

#include "ExecutionEngine.h"
#include "Compiler.h"

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

	switch (exitCode)
	{
	case 101:
		BOOST_THROW_EXCEPTION(BadJumpDestination());
	case 102:
		BOOST_THROW_EXCEPTION(OutOfGas());
	case 103:
		BOOST_THROW_EXCEPTION(StackTooSmall(1,0));
	}

	m_output = std::move(engine.returnData);
	return {m_output.data(), m_output.size()};	// TODO: This all bytesConstRef stuff sucks
}

}
}
}
