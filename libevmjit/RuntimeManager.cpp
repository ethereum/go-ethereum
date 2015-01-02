
#include "RuntimeManager.h"

#include <llvm/IR/GlobalVariable.h>
#include <llvm/IR/Function.h>
#include <llvm/IR/IntrinsicInst.h>

#include "RuntimeData.h"
#include "Instruction.h"

namespace dev
{
namespace eth
{
namespace jit
{

llvm::StructType* RuntimeManager::getRuntimeDataType()
{
	static llvm::StructType* type = nullptr;
	if (!type)
	{
		llvm::Type* elems[] =
		{
			llvm::ArrayType::get(Type::Word, RuntimeData::_size),	// i256[]
			Type::BytePtr,		// callData
			Type::BytePtr		// code
		};
		type = llvm::StructType::create(elems, "RuntimeData");
	}
	return type;
}

llvm::StructType* RuntimeManager::getRuntimeType()
{
	static llvm::StructType* type = nullptr;
	if (!type)
	{
		llvm::Type* elems[] =
		{
			Type::RuntimeDataPtr,	// data
			Type::EnvPtr,			// Env*
			Type::BytePtr,			// jmpbuf
			Type::BytePtr,			// memory data
			Type::Word,				// memory size
		};
		type = llvm::StructType::create(elems, "Runtime");
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

RuntimeManager::RuntimeManager(llvm::IRBuilder<>& _builder): CompilerHelper(_builder)
{
	m_longjmp = llvm::Intrinsic::getDeclaration(getModule(), llvm::Intrinsic::longjmp);

	// Unpack data
	auto rtPtr = getRuntimePtr();
	m_dataPtr = m_builder.CreateLoad(m_builder.CreateStructGEP(rtPtr, 0), "data");
	assert(m_dataPtr->getType() == Type::RuntimeDataPtr);
	m_envPtr = m_builder.CreateLoad(m_builder.CreateStructGEP(rtPtr, 1), "env");
	assert(m_envPtr->getType() == Type::EnvPtr);
}

llvm::Value* RuntimeManager::getRuntimePtr()
{
	// Expect first argument of a function to be a pointer to Runtime
	auto func = m_builder.GetInsertBlock()->getParent();
	auto rtPtr = &func->getArgumentList().front();
	assert(rtPtr->getType() == Type::RuntimePtr);
	return rtPtr;
}

llvm::Value* RuntimeManager::getDataPtr()
{
	if (getMainFunction())
		return m_dataPtr;

	auto rtPtr = getRuntimePtr();
	return m_builder.CreateLoad(m_builder.CreateStructGEP(rtPtr, 0), "data");
}

llvm::Value* RuntimeManager::getEnvPtr()
{
	assert(getMainFunction());	// Available only in main function
	return m_envPtr;
}

llvm::Value* RuntimeManager::getPtr(RuntimeData::Index _index)
{
	llvm::Value* idxList[] = {m_builder.getInt32(0), m_builder.getInt32(0), m_builder.getInt32(_index)};
	return m_builder.CreateInBoundsGEP(getDataPtr(), idxList, getName(_index) + "Ptr");
}

llvm::Value* RuntimeManager::get(RuntimeData::Index _index)
{
	return m_builder.CreateLoad(getPtr(_index), getName(_index));
}

void RuntimeManager::set(RuntimeData::Index _index, llvm::Value* _value)
{
	m_builder.CreateStore(_value, getPtr(_index));
}

void RuntimeManager::registerReturnData(llvm::Value* _offset, llvm::Value* _size)
{
	set(RuntimeData::ReturnDataOffset, _offset);
	set(RuntimeData::ReturnDataSize, _size);
}

void RuntimeManager::registerSuicide(llvm::Value* _balanceAddress)
{
	set(RuntimeData::SuicideDestAddress, _balanceAddress);
}

void RuntimeManager::raiseException(ReturnCode _returnCode)
{
	m_builder.CreateCall2(m_longjmp, getJmpBuf(), Constant::get(_returnCode));
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
	auto ptr = getBuilder().CreateStructGEP(getDataPtr(), 1, "calldataPtr");
	return getBuilder().CreateLoad(ptr, "calldata");
}

llvm::Value* RuntimeManager::getCode()
{
	auto ptr = getBuilder().CreateStructGEP(getDataPtr(), 2, "codePtr");
	return getBuilder().CreateLoad(ptr, "code");
}

llvm::Value* RuntimeManager::getJmpBuf()
{
	auto ptr = getBuilder().CreateStructGEP(getRuntimePtr(), 2, "jmpbufPtr");
	return getBuilder().CreateLoad(ptr, "jmpbuf");
}

llvm::Value* RuntimeManager::getGas()
{
	return get(RuntimeData::Gas);
}

void RuntimeManager::setGas(llvm::Value* _gas)
{
	llvm::Value* idxList[] = {m_builder.getInt32(0), m_builder.getInt32(0), m_builder.getInt32(RuntimeData::Gas)};
	auto ptr = m_builder.CreateInBoundsGEP(getDataPtr(), idxList, "gasPtr");
	m_builder.CreateStore(_gas, ptr);
}

}
}
}
