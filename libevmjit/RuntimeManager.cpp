
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
			Type::BytePtr			// jmpbuf
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
	m_rtPtr = new llvm::GlobalVariable(*getModule(), Type::RuntimePtr, false, llvm::GlobalVariable::PrivateLinkage, llvm::UndefValue::get(Type::RuntimePtr), "rt");
	m_dataPtr = new llvm::GlobalVariable(*getModule(), Type::RuntimeDataPtr, false, llvm::GlobalVariable::PrivateLinkage, llvm::UndefValue::get(Type::RuntimeDataPtr), "data");
	m_longjmp = llvm::Intrinsic::getDeclaration(getModule(), llvm::Intrinsic::longjmp);

	// Export data
	auto mainFunc = getMainFunction();
	llvm::Value* rtPtr = &mainFunc->getArgumentList().back();
	m_builder.CreateStore(rtPtr, m_rtPtr);
	auto dataPtr = m_builder.CreateStructGEP(rtPtr, 0, "dataPtr");
	auto data = m_builder.CreateLoad(dataPtr, "data");
	m_builder.CreateStore(data, m_dataPtr);

	auto envPtr = m_builder.CreateStructGEP(rtPtr, 1, "envPtr");
	m_env = m_builder.CreateLoad(envPtr, "env");
	assert(m_env->getType() == Type::EnvPtr);
}

llvm::Value* RuntimeManager::getRuntimePtr()
{
	// FIXME: Data ptr
	//if (auto mainFunc = getMainFunction())
	//	return mainFunc->arg_begin()->getNextNode();    // Runtime is the second parameter of main function
	return m_builder.CreateLoad(m_rtPtr, "rt");
}

llvm::Value* RuntimeManager::getDataPtr()
{
	// FIXME: Data ptr
	//if (auto mainFunc = getMainFunction())
	//	return mainFunc->arg_begin()->getNextNode();    // Runtime is the second parameter of main function
	return m_builder.CreateLoad(m_dataPtr, "data");
}

llvm::Value* RuntimeManager::getEnv()
{
	assert(getMainFunction());	// Available only in main function
	return m_env;
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
