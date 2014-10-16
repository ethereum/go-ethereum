
#pragma once

#include <libdevcore/Common.h>
#include <libevm/ExtVMFace.h>

namespace dev
{
namespace eth
{
namespace jit
{

class VM
{
public:
	/// Construct VM object.
	explicit VM(u256 _gas = 0): m_gas(_gas) {}

	void reset(u256 _gas = 0) { m_gas = _gas; }

	bytes go(ExtVMFace& _ext);

	u256 gas() const { return m_gas; }

private:
	u256 m_gas = 0;
};

}
}
}
