
#include "Ext.h"

#include <llvm/IR/Function.h>
#include <llvm/IR/TypeBuilder.h>

#include "Utils.h"

#ifdef _MSC_VER
#define EXPORT __declspec(dllexport)
#else
#define EXPORT
#endif

using namespace llvm;
using llvm::types::i;
using Linkage = llvm::GlobalValue::LinkageTypes;

namespace evmcc
{

// TODO: Copy of dev::eth::fromAddress in VM.h
inline dev::u256 fromAddress(dev::Address _a)
{
	return (dev::u160)_a;
}

std::unique_ptr<dev::eth::ExtVMFace> g_ext;

void Ext::init(std::unique_ptr<dev::eth::ExtVMFace> _ext)
{
	g_ext = std::move(_ext);
}

struct ExtData
{
	i256 address;
	i256 caller;
	i256 origin;
	i256 callvalue;
	i256 gasprice;
	i256 calldatasize;
	const byte* calldata;
};

Ext::Ext(llvm::IRBuilder<>& _builder)
	: m_builder(_builder)
{
	auto module = m_builder.GetInsertBlock()->getParent()->getParent();
	auto&& ctx = _builder.getContext();

	auto i256Ty = m_builder.getIntNTy(256);
	m_args[0] = m_builder.CreateAlloca(i256Ty, nullptr, "ext.index");
	m_args[1] = m_builder.CreateAlloca(i256Ty, nullptr, "ext.value");

	Type* elements[] = {
		i256Ty,
		i256Ty,
		i256Ty,
		i256Ty,
		i256Ty,
		i256Ty,
		m_builder.getInt8PtrTy()
	};
	auto extDataTy = StructType::create(elements, "ext.Data");

	m_data = m_builder.CreateAlloca(extDataTy, nullptr, "ext.data");

	m_init = Function::Create(FunctionType::get(m_builder.getVoidTy(), extDataTy->getPointerTo(), false), Linkage::ExternalLinkage, "ext_init", module);
	m_store = Function::Create(TypeBuilder<void(i<256>*, i<256>*), true>::get(ctx), Linkage::ExternalLinkage, "ext_store", module);
	m_setStore = Function::Create(TypeBuilder<void(i<256>*, i<256>*), true>::get(ctx), Linkage::ExternalLinkage, "ext_setStore", module);

	m_builder.CreateCall(m_init, m_data);
}

llvm::Value* Ext::store(llvm::Value* _index)
{
	m_builder.CreateStore(_index, m_args[0]);
	m_builder.CreateCall(m_store, m_args);
	return m_builder.CreateLoad(m_args[1]);
}

void Ext::setStore(llvm::Value* _index, llvm::Value* _value)
{
	m_builder.CreateStore(_index, m_args[0]);
	m_builder.CreateStore(_value, m_args[1]);
	m_builder.CreateCall(m_setStore, m_args);
}

Value* Ext::getDataElem(unsigned _index, const Twine& _name)
{
	auto valuePtr = m_builder.CreateStructGEP(m_data, _index, _name);
	return m_builder.CreateLoad(valuePtr);
}

Value* Ext::address() { return getDataElem(0, "address"); }
Value* Ext::caller() { return getDataElem(1, "caller"); }
Value* Ext::origin() { return getDataElem(2, "origin"); }
Value* Ext::callvalue() { return getDataElem(3, "callvalue"); }
Value* Ext::calldatasize() { return getDataElem(5, "calldatasize"); }
Value* Ext::gasprice() { return getDataElem(4, "gasprice"); }

extern "C"
{

EXPORT void ext_init(ExtData* _extData)
{
	_extData->address = eth2llvm(fromAddress(g_ext->myAddress));
	_extData->caller = eth2llvm(fromAddress(g_ext->caller));
	_extData->origin = eth2llvm(fromAddress(g_ext->origin));
	_extData->callvalue = eth2llvm(g_ext->value);
	_extData->gasprice = eth2llvm(g_ext->gasPrice);
	_extData->calldatasize = eth2llvm(g_ext->data.size());
	_extData->calldata = g_ext->data.data();
}

EXPORT void ext_store(i256* _index, i256* _value)
{
	auto index = llvm2eth(*_index);
	auto value = g_ext->store(index);
	*_value = eth2llvm(value);
}

EXPORT void ext_setStore(i256* _index, i256* _value)
{
	auto index = llvm2eth(*_index);
	auto value = llvm2eth(*_value);
	g_ext->setStore(index, value);
}

}

}