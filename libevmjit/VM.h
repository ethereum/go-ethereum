
#pragma once

#include <libdevcore/Common.h>
#include <libevm/VMFace.h>
#include <libevm/ExtVMFace.h>

namespace dev
{
namespace eth
{
namespace jit
{

class VM: public VMFace
{
	virtual bytesConstRef go(ExtVMFace& _ext, OnOpFunc const& _onOp = {}, uint64_t _steps = (uint64_t)-1) override final;

private:
	friend VMFace;
	explicit VM(u256 _gas = 0): VMFace(_gas) {}

	bytes m_output;
};

}
}
}
