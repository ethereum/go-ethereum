
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
			Type::i256,
		};
		type = llvm::StructType::create(elems, "RuntimeData");
	}
	return type;
}

static Runtime* g_runtime;	// FIXME: Remove

Runtime::Runtime(u256 _gas, ExtVMFace& _ext):
	m_ext(_ext)
{
	assert(!g_runtime);
	g_runtime = this;
	m_data.gas = eth2llvm(_gas);
}

Runtime::~Runtime()
{
	g_runtime = nullptr;
}


ExtVMFace& Runtime::getExt()
{
	return g_runtime->m_ext;
}

u256 Runtime::getGas()
{
	return llvm2eth(m_data.gas);
}

extern "C" {
	EXPORT i256 mem_returnDataOffset;	// FIXME: Dis-globalize
	EXPORT i256 mem_returnDataSize;
}

bytesConstRef Runtime::getReturnData()
{
	// TODO: Handle large indexes
	auto offset = static_cast<size_t>(llvm2eth(mem_returnDataOffset));
	auto size = static_cast<size_t>(llvm2eth(mem_returnDataSize));
	return{getMemory().data() + offset, size};
}


RuntimeManager::RuntimeManager(llvm::IRBuilder<>& _builder): CompilerHelper(_builder)
{
	auto dataPtrType = RuntimeData::getType()->getPointerTo();
	m_dataPtr = new llvm::GlobalVariable(*getModule(), dataPtrType, false, llvm::GlobalVariable::PrivateLinkage, llvm::UndefValue::get(dataPtrType), "rt");

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

llvm::Value* RuntimeManager::getGas()
{
	auto gasPtr = m_builder.CreateStructGEP(getRuntimePtr(), 0);
	return m_builder.CreateLoad(gasPtr, "gas");
}

void RuntimeManager::setGas(llvm::Value* _gas)
{
	auto gasPtr = m_builder.CreateStructGEP(getRuntimePtr(), 0);
	m_builder.CreateStore(_gas, gasPtr);
}

}
}
}
