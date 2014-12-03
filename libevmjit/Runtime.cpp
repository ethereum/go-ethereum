
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

Runtime::Runtime(u256 _gas, ExtVMFace& _ext, jmp_buf _jmpBuf, bool _outputLogs):
	m_ext(_ext),
	m_outputLogs(_outputLogs)
{
	set(RuntimeData::Gas, _gas);
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
	m_data.callData = _ext.data.data();
	m_data.code = _ext.code.data();
	m_data.jmpBuf = _jmpBuf;
}

void Runtime::set(RuntimeData::Index _index, u256 _value)
{
	m_data.elems[_index] = eth2llvm(_value);
}

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

bool Runtime::outputLogs() const
{
	return m_outputLogs;
}


}
}
}
