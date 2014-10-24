
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

static Runtime* g_runtime;

extern "C"
{
EXPORT i256 gas;
}

Runtime::Runtime(u256 _gas, ExtVMFace& _ext):
	m_ext(_ext)
{
	assert(!g_runtime);
	g_runtime = this;
	gas = eth2llvm(_gas);
}

Runtime::~Runtime()
{
	g_runtime = nullptr;
}

StackImpl& Runtime::getStack()
{
	return g_runtime->m_stack;
}

MemoryImpl& Runtime::getMemory()
{
	return g_runtime->m_memory;
}

ExtVMFace& Runtime::getExt()
{
	return g_runtime->m_ext;
}

u256 Runtime::getGas()
{
	return llvm2eth(gas);
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

}
}
}
