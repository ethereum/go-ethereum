#pragma once

#include <libevm/VMFace.h>
#include <evmjit/libevmjit/ExecutionEngine.h>

namespace dev
{
namespace eth
{

class JitVM: public VMFace
{
	virtual bytesConstRef go(ExtVMFace& _ext, OnOpFunc const& _onOp = {}, uint64_t _steps = (uint64_t)-1) override final;

	enum Kind: bool { Interpreter, JIT };
	static std::unique_ptr<VMFace> create(Kind, u256 _gas = 0);

private:
	friend class VMFactory;
	explicit JitVM(u256 _gas = 0) : VMFace(_gas) {}

	jit::RuntimeData m_data;
	jit::ExecutionEngine m_engine;
};


}
}
