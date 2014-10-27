
#include "Runtime.h"

#include <llvm/IR/GlobalVariable.h>
#include <llvm/IR/Function.h>

#include <libevm/VM.h>

#include "Type.h"

namespace dev
{
namespace eth
{
namespace jit
{

llvm::StructType* RuntimeData::getType()
{
	static llvm::StructType* type = nullptr;
	if (!type)
	{
		llvm::Type* elems[] =
		{
			llvm::ArrayType::get(Type::i256, _size),
			Type::BytePtr
		};
		type = llvm::StructType::create(elems, "RuntimeData");
	}
	return type;
}

namespace
{
llvm::Twine getName(RuntimeData::Index _index)
{
	switch (_index)
	{
	default:						return "data";
	case RuntimeData::Gas:			return "gas";
	case RuntimeData::Address:		return "address";
	case RuntimeData::Caller:		return "caller";
	case RuntimeData::Origin:		return "origin";
	case RuntimeData::CallValue:	return "callvalue";
	case RuntimeData::CallDataSize:	return "calldatasize";
	case RuntimeData::GasPrice:		return "gasprice";
	case RuntimeData::PrevHash:		return "prevhash";
	case RuntimeData::CoinBase:		return "coinbase";
	case RuntimeData::TimeStamp:	return "timestamp";
	case RuntimeData::Number:		return "number";
	case RuntimeData::Difficulty:	return "difficulty";
	case RuntimeData::GasLimit:		return "gaslimit";
	case RuntimeData::CodeSize:		return "codesize";
	}
}
}

static Runtime* g_runtime;	// FIXME: Remove

Runtime::Runtime(u256 _gas, ExtVMFace& _ext):
	m_ext(_ext)
{
	assert(!g_runtime);
	g_runtime = this;
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
	set(RuntimeData::CodeSize, _ext.code.size());	// TODO: Use constant
	m_data.callData = _ext.data.data();
}

Runtime::~Runtime()
{
	g_runtime = nullptr;
}

void Runtime::set(RuntimeData::Index _index, u256 _value)
{
	m_data.elems[_index] = eth2llvm(_value);
}


ExtVMFace& Runtime::getExt()
{
	return g_runtime->m_ext;
}

u256 Runtime::getGas() const
{
	return llvm2eth(m_data.elems[RuntimeData::Gas]);
}

extern "C" {
	EXPORT i256 mem_returnDataOffset;	// FIXME: Dis-globalize
	EXPORT i256 mem_returnDataSize;
}

bytesConstRef Runtime::getReturnData() const
{
	// TODO: Handle large indexes
	auto offset = static_cast<size_t>(llvm2eth(mem_returnDataOffset));
	auto size = static_cast<size_t>(llvm2eth(mem_returnDataSize));
	return {m_memory.data() + offset, size};
}


RuntimeManager::RuntimeManager(llvm::IRBuilder<>& _builder): CompilerHelper(_builder)
{
	m_dataPtr = new llvm::GlobalVariable(*getModule(), Type::RuntimePtr, false, llvm::GlobalVariable::PrivateLinkage, llvm::UndefValue::get(Type::RuntimePtr), "rt");

	// Export data
	auto mainFunc = getMainFunction();
	llvm::Value* dataPtr = &mainFunc->getArgumentList().back();
	m_builder.CreateStore(dataPtr, m_dataPtr);
}

llvm::Value* RuntimeManager::getRuntimePtr()
{
	// TODO: If in main function - get it from param
	return m_builder.CreateLoad(m_dataPtr);
}

llvm::Value* RuntimeManager::get(RuntimeData::Index _index)
{
	llvm::Value* idxList[] = {m_builder.getInt32(0), m_builder.getInt32(0), m_builder.getInt32(_index)};
	auto ptr = m_builder.CreateInBoundsGEP(getRuntimePtr(), idxList, getName(_index) + "Ptr");
	return m_builder.CreateLoad(ptr, getName(_index));
}

llvm::Value* RuntimeManager::get(Instruction _inst)
{
	switch (_inst)
	{
	default: assert(false); return nullptr;
	case Instruction::GAS:			return get(RuntimeData::Gas);
	case Instruction::ADDRESS:		return get(RuntimeData::Address);
	case Instruction::CALLER:		return get(RuntimeData::Caller);
	case Instruction::ORIGIN:		return get(RuntimeData::Origin);
	case Instruction::CALLVALUE:	return get(RuntimeData::CallValue);
	case Instruction::CALLDATASIZE:	return get(RuntimeData::CallDataSize);
	case Instruction::GASPRICE:		return get(RuntimeData::GasPrice);
	case Instruction::PREVHASH:		return get(RuntimeData::PrevHash);
	case Instruction::COINBASE:		return get(RuntimeData::CoinBase);
	case Instruction::TIMESTAMP:	return get(RuntimeData::TimeStamp);
	case Instruction::NUMBER:		return get(RuntimeData::Number);
	case Instruction::DIFFICULTY:	return get(RuntimeData::Difficulty);
	case Instruction::GASLIMIT:		return get(RuntimeData::GasLimit);
	case Instruction::CODESIZE:		return get(RuntimeData::CodeSize);
	}
}

llvm::Value* RuntimeManager::getCallData()
{
	auto ptr = getBuilder().CreateStructGEP(getRuntimePtr(), 1, "calldataPtr");
	return getBuilder().CreateLoad(ptr, "calldata");
}

llvm::Value* RuntimeManager::getGas()
{
	return get(RuntimeData::Gas);
}

void RuntimeManager::setGas(llvm::Value* _gas)
{
	llvm::Value* idxList[] = {m_builder.getInt32(0), m_builder.getInt32(0), m_builder.getInt32(RuntimeData::Gas)};
	auto ptr = m_builder.CreateInBoundsGEP(getRuntimePtr(), idxList, "gasPtr");
	m_builder.CreateStore(_gas, ptr);
}

}
}
}
