
#include "Ext.h"

#include <llvm/IR/Function.h>
#include <llvm/IR/TypeBuilder.h>

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

	Function::Create(TypeBuilder<void(i<256>*, i<256>*), true>::get(ctx), Linkage::ExternalLinkage, "ext_store", module);
	Function::Create(TypeBuilder<void(i<256>*, i<256>*), true>::get(ctx), Linkage::ExternalLinkage, "ext_setStore", module);
}

extern "C"
{

EXPORT void ext_store(void* _index, void* _value)
{

}

EXPORT void ext_setStore(void* _index, void* _value)
{

}

}

}