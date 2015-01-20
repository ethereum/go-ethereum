
#include "Ext.h"

#include <llvm/IR/Function.h>
#include <llvm/IR/TypeBuilder.h>
#include <llvm/IR/IntrinsicInst.h>

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
	m_funcs = decltype(m_funcs)();
	m_argAllocas = decltype(m_argAllocas)();
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
		FuncDesc{"env_blockhash", getFunctionType(Type::Void, {Type::EnvPtr, Type::WordPtr, Type::WordPtr})},
		FuncDesc{"env_extcode", getFunctionType(Type::BytePtr, {Type::EnvPtr, Type::WordPtr, Type::Size->getPointerTo()})},
		FuncDesc{"ext_calldataload", getFunctionType(Type::Void, {Type::RuntimeDataPtr, Type::WordPtr, Type::WordPtr})},
	}};

	return descs;
}

llvm::Function* createFunc(EnvFunc _id, llvm::Module* _module)
{
	auto&& desc = getEnvFuncDescs()[static_cast<size_t>(_id)];
	return llvm::Function::Create(std::get<1>(desc), llvm::Function::ExternalLinkage, std::get<0>(desc), _module);
}

llvm::Value* Ext::getArgAlloca()
{
	auto& a = m_argAllocas[m_argCounter++];
	if (!a)
	{
		// FIXME: Improve order and names
		InsertPointGuard g{getBuilder()};
		getBuilder().SetInsertPoint(getMainFunction()->front().getFirstNonPHI());
		a = getBuilder().CreateAlloca(Type::Word, nullptr, "arg");
	}
	return a;
}

llvm::Value* Ext::byPtr(llvm::Value* _value)
{
	auto a = getArgAlloca();
	getBuilder().CreateStore(_value, a);
	return a;
}

llvm::CallInst* Ext::createCall(EnvFunc _funcId, std::initializer_list<llvm::Value*> const& _args)
{
	auto& func = m_funcs[static_cast<size_t>(_funcId)];
	if (!func)
		func = createFunc(_funcId, getModule());

	m_argCounter = 0;
	return getBuilder().CreateCall(func, {_args.begin(), _args.size()});
}

llvm::Value* Ext::sload(llvm::Value* _index)
{
	auto ret = getArgAlloca();
	createCall(EnvFunc::sload, {getRuntimeManager().getEnvPtr(), byPtr(_index), ret}); // Uses native endianness
	return m_builder.CreateLoad(ret);
}

void Ext::sstore(llvm::Value* _index, llvm::Value* _value)
{
	createCall(EnvFunc::sstore, {getRuntimeManager().getEnvPtr(), byPtr(_index), byPtr(_value)}); // Uses native endianness
}

llvm::Value* Ext::calldataload(llvm::Value* _index)
{
	auto ret = getArgAlloca();
	createCall(EnvFunc::calldataload, {getRuntimeManager().getDataPtr(), byPtr(_index), ret});
	ret = m_builder.CreateLoad(ret);
	return Endianness::toNative(m_builder, ret);
}

llvm::Value* Ext::balance(llvm::Value* _address)
{
	auto address = Endianness::toBE(m_builder, _address);
	auto ret = getArgAlloca();
	createCall(EnvFunc::balance, {getRuntimeManager().getEnvPtr(), byPtr(address), ret});
	return m_builder.CreateLoad(ret);
}

llvm::Value* Ext::blockhash(llvm::Value* _number)
{
	auto hash = getArgAlloca();
	createCall(EnvFunc::blockhash, {getRuntimeManager().getEnvPtr(), byPtr(_number), hash});
	hash =  m_builder.CreateLoad(hash);
	return Endianness::toNative(getBuilder(), hash);
}

llvm::Value* Ext::create(llvm::Value*& _gas, llvm::Value* _endowment, llvm::Value* _initOff, llvm::Value* _initSize)
{
	auto gas = byPtr(_gas);
	auto ret = getArgAlloca();
	auto begin = m_memoryMan.getBytePtr(_initOff);
	auto size = m_builder.CreateTrunc(_initSize, Type::Size, "size");
	createCall(EnvFunc::create, {getRuntimeManager().getEnvPtr(), gas, byPtr(_endowment), begin, size, ret});
	_gas = m_builder.CreateLoad(gas); // Return gas
	llvm::Value* address = m_builder.CreateLoad(ret);
	address = Endianness::toNative(m_builder, address);
	return address;
}

llvm::Value* Ext::call(llvm::Value*& _gas, llvm::Value* _receiveAddress, llvm::Value* _value, llvm::Value* _inOff, llvm::Value* _inSize, llvm::Value* _outOff, llvm::Value* _outSize, llvm::Value* _codeAddress)
{
	auto gas = byPtr(_gas);
	auto receiveAddress = Endianness::toBE(m_builder, _receiveAddress);
	auto inBeg = m_memoryMan.getBytePtr(_inOff);
	auto inSize = m_builder.CreateTrunc(_inSize, Type::Size, "in.size");
	auto outBeg = m_memoryMan.getBytePtr(_outOff);
	auto outSize = m_builder.CreateTrunc(_outSize, Type::Size, "out.size");
	auto codeAddress = Endianness::toBE(m_builder, _codeAddress);
	auto ret = createCall(EnvFunc::call, {getRuntimeManager().getEnvPtr(), gas, byPtr(receiveAddress), byPtr(_value), inBeg, inSize, outBeg, outSize, byPtr(codeAddress)});
	_gas = m_builder.CreateLoad(gas); // Return gas
	return m_builder.CreateZExt(ret, Type::Word, "ret");
}

llvm::Value* Ext::sha3(llvm::Value* _inOff, llvm::Value* _inSize)
{
	auto begin = m_memoryMan.getBytePtr(_inOff);
	auto size = m_builder.CreateTrunc(_inSize, Type::Size, "size");
	auto ret = getArgAlloca();
	createCall(EnvFunc::sha3, {begin, size, ret});
	llvm::Value* hash = m_builder.CreateLoad(ret);
	hash = Endianness::toNative(m_builder, hash);
	return hash;
}

MemoryRef Ext::extcode(llvm::Value* _addr)
{
	auto addr = Endianness::toBE(m_builder, _addr);
	auto code = createCall(EnvFunc::extcode, {getRuntimeManager().getEnvPtr(), byPtr(addr), m_size});
	auto codeSize = m_builder.CreateLoad(m_size);
	auto codeSize256 = m_builder.CreateZExt(codeSize, Type::Word);
	return {code, codeSize256};
}

void Ext::log(llvm::Value* _memIdx, llvm::Value* _numBytes, std::array<llvm::Value*,4> const& _topics)
{
	auto begin = m_memoryMan.getBytePtr(_memIdx);
	auto size = m_builder.CreateTrunc(_numBytes, Type::Size, "size");
	llvm::Value* args[] = {getRuntimeManager().getEnvPtr(), begin, size, getArgAlloca(), getArgAlloca(), getArgAlloca(), getArgAlloca()};

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
