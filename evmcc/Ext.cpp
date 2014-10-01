
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

std::unique_ptr<dev::eth::ExtVMFace> g_ext;

void Ext::init(std::unique_ptr<dev::eth::ExtVMFace> _ext)
{
	g_ext = std::move(_ext);
}

Ext::Ext(llvm::IRBuilder<>& _builder)
	: m_builder(_builder)
{
	auto module = m_builder.GetInsertBlock()->getParent()->getParent();
	auto&& ctx = _builder.getContext();

	m_args[0] = m_builder.CreateAlloca(m_builder.getIntNTy(256), nullptr, "ext.index");
	m_args[1] = m_builder.CreateAlloca(m_builder.getIntNTy(256), nullptr, "ext.value");

	m_store    = Function::Create(TypeBuilder<void(i<256>*, i<256>*), true>::get(ctx), Linkage::ExternalLinkage, "ext_store", module);
	m_setStore = Function::Create(TypeBuilder<void(i<256>*, i<256>*), true>::get(ctx), Linkage::ExternalLinkage, "ext_setStore", module);
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

extern "C"
{

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