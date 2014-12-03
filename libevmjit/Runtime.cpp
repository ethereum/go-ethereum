
#include "Runtime.h"

#include <llvm/IR/GlobalVariable.h>
#include <llvm/IR/Function.h>
#include <llvm/IR/IntrinsicInst.h>

//#include <libevm/VM.h>

namespace dev
{
namespace eth
{
namespace jit
{

Runtime::Runtime(RuntimeData* _data, Env* _env, JmpBufRef _jmpBuf):
	m_data(*_data),
	m_env(*_env),
	m_jmpBuf(_jmpBuf)
{}

u256 Runtime::getGas() const
{
	return llvm2eth(m_data.elems[RuntimeData::Gas]);
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
