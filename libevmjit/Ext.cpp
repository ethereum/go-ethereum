
#include "Ext.h"

#include <llvm/IR/Function.h>
#include <llvm/IR/TypeBuilder.h>
#include <llvm/IR/IntrinsicInst.h>

//#include <libdevcrypto/SHA3.h>
//#include <libevm/FeeStructure.h>

#include "RuntimeManager.h"
#include "Memory.h"
#include "Type.h"
#include "Endianness.h"

namespace dev
{
namespace eth
{
namespace jit
{

Ext::Ext(RuntimeManager& _runtimeManager, Memory& _memoryMan):
	RuntimeHelper(_runtimeManager),
	m_memoryMan(_memoryMan)
{
	auto module = getModule();

	m_args[0] = m_builder.CreateAlloca(Type::Word, nullptr, "ext.index");
	m_args[1] = m_builder.CreateAlloca(Type::Word, nullptr, "ext.value");
	m_arg2 = m_builder.CreateAlloca(Type::Word, nullptr, "ext.arg2");
	m_arg3 = m_builder.CreateAlloca(Type::Word, nullptr, "ext.arg3");
	m_arg4 = m_builder.CreateAlloca(Type::Word, nullptr, "ext.arg4");
	m_arg5 = m_builder.CreateAlloca(Type::Word, nullptr, "ext.arg5");
	m_arg6 = m_builder.CreateAlloca(Type::Word, nullptr, "ext.arg6");
	m_arg7 = m_builder.CreateAlloca(Type::Word, nullptr, "ext.arg7");
	m_arg8 = m_builder.CreateAlloca(Type::Word, nullptr, "ext.arg8");
	m_size = m_builder.CreateAlloca(Type::Size, nullptr, "env.size");

	using Linkage = llvm::GlobalValue::LinkageTypes;

	llvm::Type* argsTypes[] = {Type::EnvPtr, Type::WordPtr, Type::WordPtr, Type::WordPtr, Type::WordPtr, Type::WordPtr, Type::WordPtr, Type::WordPtr, Type::WordPtr, Type::WordPtr};

	m_sload = llvm::Function::Create(llvm::FunctionType::get(Type::Void, {argsTypes, 3}, false), Linkage::ExternalLinkage, "env_sload", module);
	m_sstore = llvm::Function::Create(llvm::FunctionType::get(Type::Void, {argsTypes, 3}, false), Linkage::ExternalLinkage, "env_sstore", module);

	llvm::Type* sha3ArgsTypes[] = {Type::BytePtr, Type::Size, Type::WordPtr};
	m_sha3 = llvm::Function::Create(llvm::FunctionType::get(Type::Void, sha3ArgsTypes, false), Linkage::ExternalLinkage, "env_sha3", module);

	llvm::Type* createArgsTypes[] = {Type::EnvPtr, Type::WordPtr, Type::WordPtr, Type::BytePtr, Type::Size, Type::WordPtr};
	m_create = llvm::Function::Create(llvm::FunctionType::get(Type::Void, createArgsTypes, false), Linkage::ExternalLinkage, "env_create", module);

	llvm::Type* callArgsTypes[] = {Type::EnvPtr, Type::WordPtr, Type::WordPtr, Type::WordPtr, Type::BytePtr, Type::Size, Type::BytePtr, Type::Size, Type::WordPtr};
	m_call = llvm::Function::Create(llvm::FunctionType::get(Type::Bool, callArgsTypes, false), Linkage::ExternalLinkage, "env_call", module);

	llvm::Type* logArgsTypes[] = {Type::EnvPtr, Type::BytePtr, Type::Size, Type::WordPtr, Type::WordPtr, Type::WordPtr, Type::WordPtr};
	m_log = llvm::Function::Create(llvm::FunctionType::get(Type::Void, logArgsTypes, false), Linkage::ExternalLinkage, "env_log", module);

	llvm::Type* getExtCodeArgsTypes[] = {Type::EnvPtr, Type::WordPtr, Type::Size->getPointerTo()};
	m_getExtCode = llvm::Function::Create(llvm::FunctionType::get(Type::BytePtr, getExtCodeArgsTypes, false), Linkage::ExternalLinkage, "env_getExtCode", module);

	// Helper function, not client Env interface
	llvm::Type* callDataLoadArgsTypes[] = {Type::RuntimeDataPtr, Type::WordPtr, Type::WordPtr};
	m_calldataload = llvm::Function::Create(llvm::FunctionType::get(Type::Void, callDataLoadArgsTypes, false), Linkage::ExternalLinkage, "ext_calldataload", module);
}

llvm::Function* Ext::getBalanceFunc()
{
	if (!m_balance)
	{
		llvm::Type* argsTypes[] = {Type::EnvPtr, Type::WordPtr, Type::WordPtr};
		m_balance = llvm::Function::Create(llvm::FunctionType::get(Type::Void, argsTypes, false), llvm::Function::ExternalLinkage, "env_balance", getModule());
	}
	return m_balance;
}

llvm::Value* Ext::sload(llvm::Value* _index)
{
	m_builder.CreateStore(_index, m_args[0]);
	m_builder.CreateCall3(m_sload, getRuntimeManager().getEnvPtr(), m_args[0], m_args[1]); // Uses native endianness
	return m_builder.CreateLoad(m_args[1]);
}

void Ext::sstore(llvm::Value* _index, llvm::Value* _value)
{
	m_builder.CreateStore(_index, m_args[0]);
	m_builder.CreateStore(_value, m_args[1]);
	m_builder.CreateCall3(m_sstore, getRuntimeManager().getEnvPtr(), m_args[0], m_args[1]); // Uses native endianness
}

llvm::Value* Ext::calldataload(llvm::Value* _index)
{
	m_builder.CreateStore(_index, m_args[0]);
	createCall(m_calldataload, getRuntimeManager().getDataPtr(), m_args[0], m_args[1]);
	auto ret = m_builder.CreateLoad(m_args[1]);
	return Endianness::toNative(m_builder, ret);
}

llvm::Value* Ext::balance(llvm::Value* _address)
{
	auto address = Endianness::toBE(m_builder, _address);
	m_builder.CreateStore(address, m_args[0]);
	createCall(getBalanceFunc(), getRuntimeManager().getEnvPtr(), m_args[0], m_args[1]);
	return m_builder.CreateLoad(m_args[1]);
}

llvm::Value* Ext::create(llvm::Value*& _gas, llvm::Value* _endowment, llvm::Value* _initOff, llvm::Value* _initSize)
{
	m_builder.CreateStore(_gas, m_args[0]);
	m_builder.CreateStore(_endowment, m_arg2);
	auto begin = m_memoryMan.getBytePtr(_initOff);
	auto size = m_builder.CreateTrunc(_initSize, Type::Size, "size");
	createCall(m_create, getRuntimeManager().getEnvPtr(), m_args[0], m_arg2, begin, size, m_args[1]);
	_gas = m_builder.CreateLoad(m_args[0]); // Return gas
	llvm::Value* address = m_builder.CreateLoad(m_args[1]);
	address = Endianness::toNative(m_builder, address);
	return address;
}

llvm::Value* Ext::call(llvm::Value*& _gas, llvm::Value* _receiveAddress, llvm::Value* _value, llvm::Value* _inOff, llvm::Value* _inSize, llvm::Value* _outOff, llvm::Value* _outSize, llvm::Value* _codeAddress)
{
	m_builder.CreateStore(_gas, m_args[0]);
	auto receiveAddress = Endianness::toBE(m_builder, _receiveAddress);
	m_builder.CreateStore(receiveAddress, m_arg2);
	m_builder.CreateStore(_value, m_arg3);
	auto inBeg = m_memoryMan.getBytePtr(_inOff);
	auto inSize = m_builder.CreateTrunc(_inSize, Type::Size, "in.size");
	auto outBeg = m_memoryMan.getBytePtr(_outOff);
	auto outSize = m_builder.CreateTrunc(_outSize, Type::Size, "out.size");
	auto codeAddress = Endianness::toBE(m_builder, _codeAddress);
	m_builder.CreateStore(codeAddress, m_arg8);
	auto ret = createCall(m_call, getRuntimeManager().getEnvPtr(), m_args[0], m_arg2, m_arg3, inBeg, inSize, outBeg, outSize, m_arg8);
	_gas = m_builder.CreateLoad(m_args[0]); // Return gas
	return m_builder.CreateZExt(ret, Type::Word, "ret");
}

llvm::Value* Ext::sha3(llvm::Value* _inOff, llvm::Value* _inSize)
{
	auto begin = m_memoryMan.getBytePtr(_inOff);
	auto size = m_builder.CreateTrunc(_inSize, Type::Size, "size");
	createCall(m_sha3, begin, size, m_args[1]);
	llvm::Value* hash = m_builder.CreateLoad(m_args[1]);
	hash = Endianness::toNative(m_builder, hash);
	return hash;
}

MemoryRef Ext::getExtCode(llvm::Value* _addr)
{
	auto addr = Endianness::toBE(m_builder, _addr);
	m_builder.CreateStore(addr, m_args[0]);
	auto code = createCall(m_getExtCode, getRuntimeManager().getEnvPtr(), m_args[0], m_size);
	auto codeSize = m_builder.CreateLoad(m_size);
	auto codeSize256 = m_builder.CreateZExt(codeSize, Type::Word);
	return {code, codeSize256};
}

void Ext::log(llvm::Value* _memIdx, llvm::Value* _numBytes, std::array<llvm::Value*,4> const& _topics)
{
	auto begin = m_memoryMan.getBytePtr(_memIdx);
	auto size = m_builder.CreateTrunc(_numBytes, Type::Size, "size");
	llvm::Value* args[] = {getRuntimeManager().getEnvPtr(), begin, size, m_arg2, m_arg3, m_arg4, m_arg5};

	auto topicArgPtr = &args[3];
	for (auto&& topic : _topics)
	{
		if (topic)
			m_builder.CreateStore(Endianness::toBE(m_builder, topic), *topicArgPtr);
		else
			*topicArgPtr = llvm::ConstantPointerNull::get(Type::WordPtr);
		++topicArgPtr;
	}

	m_builder.CreateCall(m_log, args);
}

}
}
}
