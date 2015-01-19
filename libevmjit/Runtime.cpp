
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

Runtime::Runtime(RuntimeData* _data, Env* _env) :
	m_data(*_data),
	m_env(*_env),
	m_currJmpBuf(m_jmpBuf)
{}

bytes_ref Runtime::getReturnData() const
{
	// TODO: Handle large indexes
	auto offset = static_cast<size_t>(m_data.elems[RuntimeData::ReturnDataOffset].a);
	auto size = static_cast<size_t>(m_data.elems[RuntimeData::ReturnDataSize].a);

	assert(offset + size <= m_memory.size() || size == 0);
	if (offset + size > m_memory.size())
		return {};

	auto dataBeg = m_memory.data() + offset;
	return bytes_ref{dataBeg, size};
}

}
}
}
