#include "ExecutionEngine.h"

#include <chrono>

#include <llvm/IR/Module.h>
#include <llvm/ADT/Triple.h>
#include <llvm/ExecutionEngine/ExecutionEngine.h>
#include <llvm/ExecutionEngine/SectionMemoryManager.h>
#include <llvm/ExecutionEngine/MCJIT.h>
#include <llvm/Support/TargetSelect.h>
#include <llvm/Support/Host.h>

#include "Runtime.h"
#include "Compiler.h"
#include "Cache.h"

namespace dev
{
namespace eth
{
namespace jit
{

ReturnCode ExecutionEngine::run(bytes const& _code, RuntimeData* _data, Env* _env)
{
	auto module = Compiler({}).compile(_code);
	//module->dump();
	return run(std::move(module), _data, _env, _code);
}

namespace
{
typedef ReturnCode(*EntryFuncPtr)(Runtime*);

ReturnCode runEntryFunc(EntryFuncPtr _mainFunc, Runtime* _runtime)
{
	// That function uses long jumps to handle "execeptions".
	// Do not create any non-POD objects here

	ReturnCode returnCode{};
	auto sj = setjmp(_runtime->getJmpBuf());
	if (sj == 0)
		returnCode = _mainFunc(_runtime);
	else
		returnCode = static_cast<ReturnCode>(sj);

	return returnCode;
}
}

ReturnCode ExecutionEngine::run(std::unique_ptr<llvm::Module> _module, RuntimeData* _data, Env* _env, bytes const& _code)
{
	static std::unique_ptr<llvm::ExecutionEngine> ee;  // TODO: Use Managed Objects from LLVM?

	EntryFuncPtr entryFuncPtr{};
	auto&& mainFuncName = _module->getModuleIdentifier();
	Runtime runtime(_data, _env);	// TODO: I don't know why but it must be created before getFunctionAddress() calls

	if (!ee)
	{
		llvm::InitializeNativeTarget();
		llvm::InitializeNativeTargetAsmPrinter();

		llvm::EngineBuilder builder(_module.get());
		builder.setEngineKind(llvm::EngineKind::JIT);
		builder.setUseMCJIT(true);
		std::unique_ptr<llvm::SectionMemoryManager> memoryManager(new llvm::SectionMemoryManager);
		builder.setMCJITMemoryManager(memoryManager.get());
		builder.setOptLevel(llvm::CodeGenOpt::None);

		auto triple = llvm::Triple(llvm::sys::getProcessTriple());
		if (triple.getOS() == llvm::Triple::OSType::Win32)
			triple.setObjectFormat(llvm::Triple::ObjectFormatType::ELF);  // MCJIT does not support COFF format
		_module->setTargetTriple(triple.str());

		ee.reset(builder.create());
		if (!ee)
			return ReturnCode::LLVMConfigError;

		_module.release();        // Successfully created llvm::ExecutionEngine takes ownership of the module
		memoryManager.release();  // and memory manager

		//ee->setObjectCache(Cache::getObjectCache());
		entryFuncPtr = (EntryFuncPtr)ee->getFunctionAddress(mainFuncName);
	}
	else
	{
		entryFuncPtr = (EntryFuncPtr)ee->getFunctionAddress(mainFuncName);
		if (!entryFuncPtr)
		{
			ee->addModule(_module.get());
			_module.release();
			entryFuncPtr = (EntryFuncPtr)ee->getFunctionAddress(mainFuncName);
		}
	}
	assert(entryFuncPtr);

	auto executionStartTime = std::chrono::high_resolution_clock::now();

	auto returnCode = runEntryFunc(entryFuncPtr, &runtime);
	if (returnCode == ReturnCode::Return)
		this->returnData = runtime.getReturnData();

	auto executionEndTime = std::chrono::high_resolution_clock::now();
	clog(JIT) << " + " << std::chrono::duration_cast<std::chrono::milliseconds>(executionEndTime - executionStartTime).count() << " ms\n";

	return returnCode;
}

}
}
}
