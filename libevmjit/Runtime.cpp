
#include "Runtime.h"

#include <llvm/IR/GlobalVariable.h>
#include <llvm/IR/Function.h>
#include <llvm/IR/IntrinsicInst.h>

namespace dev
{
namespace eth
{
namespace jit
{
namespace
{
	jmp_buf_ref g_currJmpBuf;
}

jmp_buf_ref Runtime::getCurrJmpBuf()
{
	return g_currJmpBuf;
}

Runtime::Runtime(RuntimeData* _data, Env* _env):
	m_data(*_data),
	m_env(*_env),
	m_currJmpBuf(m_jmpBuf),
	m_prevJmpBuf(g_currJmpBuf)
{
	g_currJmpBuf = m_jmpBuf;
}

Runtime::~Runtime()
{
	g_currJmpBuf = m_prevJmpBuf;
}

bytes Runtime::getReturnData() const	// FIXME: Reconsider returning by copy
{
	// TODO: Handle large indexes
	auto offset = static_cast<size_t>(llvm2eth(m_data.elems[RuntimeData::ReturnDataOffset]));
	auto size = static_cast<size_t>(llvm2eth(m_data.elems[RuntimeData::ReturnDataSize]));

	assert(offset + size <= m_memory.size());
	// TODO: Handle invalid data access by returning empty ref
	auto dataBeg = m_memory.begin() + offset;
	return {dataBeg, dataBeg + size};
}

}
}
}
