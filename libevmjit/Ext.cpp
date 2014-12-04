
#include "Ext.h"

#include <llvm/IR/Function.h>
#include <llvm/IR/TypeBuilder.h>
#include <llvm/IR/IntrinsicInst.h>

//#include <libdevcrypto/SHA3.h>
//#include <libevm/FeeStructure.h>

#include "RuntimeManager.h"
#include "Type.h"
#include "Endianness.h"

namespace dev
{
namespace eth
{
namespace jit
{

Ext::Ext(RuntimeManager& _runtimeManager):
	RuntimeHelper(_runtimeManager),
	m_data()
{
	auto&& ctx = m_builder.getContext();
	auto module = getModule();

	auto i256Ty = m_builder.getIntNTy(256);

	m_args[0] = m_builder.CreateAlloca(i256Ty, nullptr, "ext.index");
	m_args[1] = m_builder.CreateAlloca(i256Ty, nullptr, "ext.value");
	m_arg2 = m_builder.CreateAlloca(i256Ty, nullptr, "ext.arg2");
	m_arg3 = m_builder.CreateAlloca(i256Ty, nullptr, "ext.arg3");
	m_arg4 = m_builder.CreateAlloca(i256Ty, nullptr, "ext.arg4");
	m_arg5 = m_builder.CreateAlloca(i256Ty, nullptr, "ext.arg5");
	m_arg6 = m_builder.CreateAlloca(i256Ty, nullptr, "ext.arg6");
	m_arg7 = m_builder.CreateAlloca(i256Ty, nullptr, "ext.arg7");
	m_arg8 = m_builder.CreateAlloca(i256Ty, nullptr, "ext.arg8");

	using Linkage = llvm::GlobalValue::LinkageTypes;

	llvm::Type* argsTypes[] = {Type::EnvPtr, Type::WordPtr, Type::WordPtr, Type::WordPtr, Type::WordPtr, Type::WordPtr, Type::WordPtr, Type::WordPtr, Type::WordPtr, Type::WordPtr};

	m_store = llvm::Function::Create(llvm::FunctionType::get(Type::Void, {argsTypes, 3}, false), Linkage::ExternalLinkage, "ext_store", module);
	m_setStore = llvm::Function::Create(llvm::FunctionType::get(Type::Void, {argsTypes, 3}, false), Linkage::ExternalLinkage, "ext_setStore", module);
	m_calldataload = llvm::Function::Create(llvm::FunctionType::get(Type::Void, {argsTypes, 3}, false), Linkage::ExternalLinkage, "ext_calldataload", module);
	m_balance = llvm::Function::Create(llvm::FunctionType::get(Type::Void, {argsTypes, 3}, false), Linkage::ExternalLinkage, "ext_balance", module);
	m_suicide = llvm::Function::Create(llvm::FunctionType::get(Type::Void, {argsTypes, 2}, false), Linkage::ExternalLinkage, "ext_suicide", module);
	m_create = llvm::Function::Create(llvm::FunctionType::get(Type::Void, {argsTypes, 5}, false), Linkage::ExternalLinkage, "ext_create", module);
	m_call = llvm::Function::Create(llvm::FunctionType::get(Type::Void, argsTypes, false), Linkage::ExternalLinkage, "ext_call", module);
	m_sha3 = llvm::Function::Create(llvm::FunctionType::get(Type::Void, {argsTypes, 4}, false), Linkage::ExternalLinkage, "ext_sha3", module);
	m_exp = llvm::Function::Create(llvm::FunctionType::get(Type::Void, {argsTypes, 4}, false), Linkage::ExternalLinkage, "ext_exp", module);
	m_codeAt = llvm::Function::Create(llvm::FunctionType::get(Type::BytePtr, {argsTypes, 2}, false), Linkage::ExternalLinkage, "ext_codeAt", module);
	m_codesizeAt = llvm::Function::Create(llvm::FunctionType::get(Type::Void, {argsTypes, 3}, false), Linkage::ExternalLinkage, "ext_codesizeAt", module);
	m_log0 = llvm::Function::Create(llvm::FunctionType::get(Type::Void, {argsTypes, 3}, false), Linkage::ExternalLinkage, "ext_log0", module);
	m_log1 = llvm::Function::Create(llvm::FunctionType::get(Type::Void, {argsTypes, 4}, false), Linkage::ExternalLinkage, "ext_log1", module);
	m_log2 = llvm::Function::Create(llvm::FunctionType::get(Type::Void, {argsTypes, 5}, false), Linkage::ExternalLinkage, "ext_log2", module);
	m_log3 = llvm::Function::Create(llvm::FunctionType::get(Type::Void, {argsTypes, 6}, false), Linkage::ExternalLinkage, "ext_log3", module);
	m_log4 = llvm::Function::Create(llvm::FunctionType::get(Type::Void, {argsTypes, 7}, false), Linkage::ExternalLinkage, "ext_log4", module);
}

llvm::Value* Ext::store(llvm::Value* _index)
{
	m_builder.CreateStore(_index, m_args[0]);
	m_builder.CreateCall3(m_store, getRuntimeManager().getEnv(), m_args[0], m_args[1]); // Uses native endianness
	return m_builder.CreateLoad(m_args[1]);
}

void Ext::setStore(llvm::Value* _index, llvm::Value* _value)
{
	m_builder.CreateStore(_index, m_args[0]);
	m_builder.CreateStore(_value, m_args[1]);
	m_builder.CreateCall3(m_setStore, getRuntimeManager().getEnv(), m_args[0], m_args[1]); // Uses native endianness
}

llvm::Value* Ext::calldataload(llvm::Value* _index)
{
	m_builder.CreateStore(_index, m_args[0]);
	m_builder.CreateCall3(m_calldataload, getRuntimeManager().getEnv(), m_args[0], m_args[1]);
	auto ret = m_builder.CreateLoad(m_args[1]);
	return Endianness::toNative(m_builder, ret);
}

llvm::Value* Ext::balance(llvm::Value* _address)
{
	auto address = Endianness::toBE(m_builder, _address);
	m_builder.CreateStore(address, m_args[0]);
	m_builder.CreateCall3(m_balance, getRuntimeManager().getEnv(), m_args[0], m_args[1]);
	return m_builder.CreateLoad(m_args[1]);
}

void Ext::suicide(llvm::Value* _address)
{
	auto address = Endianness::toBE(m_builder, _address);
	m_builder.CreateStore(address, m_args[0]);
	m_builder.CreateCall2(m_suicide, getRuntimeManager().getEnv(), m_args[0]);
}

llvm::Value* Ext::create(llvm::Value* _endowment, llvm::Value* _initOff, llvm::Value* _initSize)
{
	m_builder.CreateStore(_endowment, m_args[0]);
	m_builder.CreateStore(_initOff, m_arg2);
	m_builder.CreateStore(_initSize, m_arg3);
	createCall(m_create, getRuntimeManager().getRuntimePtr(), m_args[0], m_arg2, m_arg3, m_args[1]);
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
	m_builder.CreateStore(_inOff, m_arg4);
	m_builder.CreateStore(_inSize, m_arg5);
	m_builder.CreateStore(_outOff, m_arg6);
	m_builder.CreateStore(_outSize, m_arg7);
	auto codeAddress = Endianness::toBE(m_builder, _codeAddress);
	m_builder.CreateStore(codeAddress, m_arg8);
	createCall(m_call, getRuntimeManager().getEnv(), m_args[0], m_arg2, m_arg3, m_arg4, m_arg5, m_arg6, m_arg7, m_arg8, m_args[1]);
	_gas = m_builder.CreateLoad(m_args[0]); // Return gas
	return m_builder.CreateLoad(m_args[1]);
}

llvm::Value* Ext::sha3(llvm::Value* _inOff, llvm::Value* _inSize)
{
	m_builder.CreateStore(_inOff, m_args[0]);
	m_builder.CreateStore(_inSize, m_arg2);
	createCall(m_sha3, getRuntimeManager().getEnv(), m_args[0], m_arg2, m_args[1]);
	llvm::Value* hash = m_builder.CreateLoad(m_args[1]);
	hash = Endianness::toNative(m_builder, hash);
	return hash;
}

llvm::Value* Ext::codeAt(llvm::Value* _addr)
{
	auto addr = Endianness::toBE(m_builder, _addr);
	m_builder.CreateStore(addr, m_args[0]);
	return m_builder.CreateCall2(m_codeAt, getRuntimeManager().getEnv(), m_args[0]);
}

llvm::Value* Ext::codesizeAt(llvm::Value* _addr)
{
	auto addr = Endianness::toBE(m_builder, _addr);
	m_builder.CreateStore(addr, m_args[0]);
	createCall(m_codesizeAt, getRuntimeManager().getEnv(), m_args[0], m_args[1]);
	return m_builder.CreateLoad(m_args[1]);
}

void Ext::log(llvm::Value* _memIdx, llvm::Value* _numBytes, size_t _numTopics, std::array<llvm::Value*,4> const& _topics)
{
	llvm::Value* args[] = {nullptr, m_args[0], m_args[1], m_arg2, m_arg3, m_arg4, m_arg5};
	llvm::Value* funcs[] = {m_log0, m_log1, m_log2, m_log3, m_log4};

	args[0] = getRuntimeManager().getEnv();
	m_builder.CreateStore(_memIdx, m_args[0]);
	m_builder.CreateStore(_numBytes, m_args[1]);

	for (size_t i = 0; i < _numTopics; ++i)
		m_builder.CreateStore(_topics[i], args[i + 3]);

	m_builder.CreateCall(funcs[_numTopics], llvm::ArrayRef<llvm::Value*>(args, _numTopics + 3));
}

}
}
}
