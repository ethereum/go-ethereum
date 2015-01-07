
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
}


using FuncDesc = std::tuple<char const*, llvm::FunctionType*>;

llvm::FunctionType* getFunctionType(llvm::Type* _returnType, std::initializer_list<llvm::Type*> const& _argsTypes)
{
	return llvm::FunctionType::get(_returnType, llvm::ArrayRef<llvm::Type*>{_argsTypes.begin(), _argsTypes.size()}, false);
}

std::array<FuncDesc, sizeOf<EnvFunc>::value> const& getEnvFuncDescs()
{
	static std::array<FuncDesc, sizeOf<EnvFunc>::value> descs{{
		FuncDesc{"env_sload",   getFunctionType(Type::Void, {Type::EnvPtr, Type::WordPtr, Type::WordPtr})},
		FuncDesc{"env_sstore",  getFunctionType(Type::Void, {Type::EnvPtr, Type::WordPtr, Type::WordPtr})},
		FuncDesc{"env_sha3", getFunctionType(Type::Void, {Type::BytePtr, Type::Size, Type::WordPtr})},
		FuncDesc{"env_balance", getFunctionType(Type::Void, {Type::EnvPtr, Type::WordPtr, Type::WordPtr})},
		FuncDesc{"env_create", getFunctionType(Type::Void, {Type::EnvPtr, Type::WordPtr, Type::WordPtr, Type::BytePtr, Type::Size, Type::WordPtr})},
		FuncDesc{"env_call", getFunctionType(Type::Bool, {Type::EnvPtr, Type::WordPtr, Type::WordPtr, Type::WordPtr, Type::BytePtr, Type::Size, Type::BytePtr, Type::Size, Type::WordPtr})},
		FuncDesc{"env_log", getFunctionType(Type::Void, {Type::EnvPtr, Type::BytePtr, Type::Size, Type::WordPtr, Type::WordPtr, Type::WordPtr, Type::WordPtr})},
		FuncDesc{"env_getExtCode", getFunctionType(Type::BytePtr, {Type::EnvPtr, Type::WordPtr, Type::Size->getPointerTo()})},
		FuncDesc{"ext_calldataload", getFunctionType(Type::Void, {Type::RuntimeDataPtr, Type::WordPtr, Type::WordPtr})},
	}};

	return descs;
}

llvm::Function* createFunc(EnvFunc _id, llvm::Module* _module)
{
	auto&& desc = getEnvFuncDescs()[static_cast<size_t>(_id)];
	return llvm::Function::Create(std::get<1>(desc), llvm::Function::ExternalLinkage, std::get<0>(desc), _module);
}

llvm::CallInst* Ext::createCall(EnvFunc _funcId, std::initializer_list<llvm::Value*> const& _args)
{
	auto& func = m_funcs[static_cast<size_t>(_funcId)];
	if (!func)
		func = createFunc(_funcId, getModule());

	return getBuilder().CreateCall(func, {_args.begin(), _args.size()});
}

llvm::Value* Ext::sload(llvm::Value* _index)
{
	m_builder.CreateStore(_index, m_args[0]);
	createCall(EnvFunc::sload, {getRuntimeManager().getEnvPtr(), m_args[0], m_args[1]}); // Uses native endianness
	return m_builder.CreateLoad(m_args[1]);
}

void Ext::sstore(llvm::Value* _index, llvm::Value* _value)
{
	m_builder.CreateStore(_index, m_args[0]);
	m_builder.CreateStore(_value, m_args[1]);
	createCall(EnvFunc::sstore, {getRuntimeManager().getEnvPtr(), m_args[0], m_args[1]}); // Uses native endianness
}

llvm::Value* Ext::calldataload(llvm::Value* _index)
{
	m_builder.CreateStore(_index, m_args[0]);
	createCall(EnvFunc::calldataload, {getRuntimeManager().getDataPtr(), m_args[0], m_args[1]});
	auto ret = m_builder.CreateLoad(m_args[1]);
	return Endianness::toNative(m_builder, ret);
}

llvm::Value* Ext::balance(llvm::Value* _address)
{
	auto address = Endianness::toBE(m_builder, _address);
	m_builder.CreateStore(address, m_args[0]);
	createCall(EnvFunc::balance, {getRuntimeManager().getEnvPtr(), m_args[0], m_args[1]});
	return m_builder.CreateLoad(m_args[1]);
}

llvm::Value* Ext::create(llvm::Value*& _gas, llvm::Value* _endowment, llvm::Value* _initOff, llvm::Value* _initSize)
{
	m_builder.CreateStore(_gas, m_args[0]);
	m_builder.CreateStore(_endowment, m_arg2);
	auto begin = m_memoryMan.getBytePtr(_initOff);
	auto size = m_builder.CreateTrunc(_initSize, Type::Size, "size");
	createCall(EnvFunc::create, {getRuntimeManager().getEnvPtr(), m_args[0], m_arg2, begin, size, m_args[1]});
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
	auto ret = createCall(EnvFunc::call, {getRuntimeManager().getEnvPtr(), m_args[0], m_arg2, m_arg3, inBeg, inSize, outBeg, outSize, m_arg8});
	_gas = m_builder.CreateLoad(m_args[0]); // Return gas
	return m_builder.CreateZExt(ret, Type::Word, "ret");
}

llvm::Value* Ext::sha3(llvm::Value* _inOff, llvm::Value* _inSize)
{
	auto begin = m_memoryMan.getBytePtr(_inOff);
	auto size = m_builder.CreateTrunc(_inSize, Type::Size, "size");
	createCall(EnvFunc::sha3, {begin, size, m_args[1]});
	llvm::Value* hash = m_builder.CreateLoad(m_args[1]);
	hash = Endianness::toNative(m_builder, hash);
	return hash;
}

MemoryRef Ext::getExtCode(llvm::Value* _addr)
{
	auto addr = Endianness::toBE(m_builder, _addr);
	m_builder.CreateStore(addr, m_args[0]);
	auto code = createCall(EnvFunc::getExtCode, {getRuntimeManager().getEnvPtr(), m_args[0], m_size});
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

	createCall(EnvFunc::log, {args[0], args[1], args[2], args[3], args[4], args[5], args[6]});  // TODO: use std::initializer_list<>
}

}
}
}
