#include "ExecutionEngine.h"

#include <chrono>

#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wunused-parameter"

#include <llvm/IR/LLVMContext.h>
#include <llvm/IR/Module.h>
#include <llvm/ADT/Triple.h>
#include <llvm/ExecutionEngine/ExecutionEngine.h>
#include <llvm/ExecutionEngine/SectionMemoryManager.h>
#include <llvm/ExecutionEngine/GenericValue.h>
#include <llvm/ExecutionEngine/MCJIT.h>
#include <llvm/Support/TargetSelect.h>
#include <llvm/Support/Signals.h>
#include <llvm/Support/PrettyStackTrace.h>
#include <llvm/Support/Host.h>

#pragma GCC diagnostic pop

#include "Runtime.h"
#include "Memory.h"
#include "Stack.h"
#include "Type.h"
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
	std::string key{reinterpret_cast<char const*>(_code.data()), _code.size()};
	/*if (auto cachedExec = Cache::findExec(key))
	{
		return run(*cachedExec, _data, _env);
	}*/

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
	// TODO: Use it in evmcc
	//llvm::sys::PrintStackTraceOnErrorSignal();
	//static const auto program = "EVM JIT";
	//llvm::PrettyStackTraceProgram X(1, &program);

	static std::unique_ptr<llvm::ExecutionEngine> ee;  // TODO: Use Managed Objects from LLVM?

	typedef ReturnCode(*EntryFuncPtr)(Runtime*);
	EntryFuncPtr entryFuncPtr{};


	auto&& mainFuncName = _module->getModuleIdentifier();

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
	}
	else
	{
		if (entryFuncPtr = (EntryFuncPtr)ee->getFunctionAddress(_module->getModuleIdentifier()))
		{
			entryFuncPtr = nullptr;
		}
		else
		{
			ee->addModule(_module.get());
			//std::cerr << _module->getModuleIdentifier() << "\n";
			_module.release();
		}
	}

	assert(ee);

	//ExecBundle exec;
	//exec.engine.reset(builder.create());
	//if (!exec.engine)
	//	return ReturnCode::LLVMConfigError;

	// TODO: Finalization not needed when llvm::ExecutionEngine::getFunctionAddress used
	//auto finalizationStartTime = std::chrono::high_resolution_clock::now();
	//exec.engine->finalizeObject();
	//auto finalizationEndTime = std::chrono::high_resolution_clock::now();
	//clog(JIT) << " + " << std::chrono::duration_cast<std::chrono::milliseconds>(finalizationEndTime - finalizationStartTime).count();

	auto executionStartTime = std::chrono::high_resolution_clock::now();

	std::string key{reinterpret_cast<char const*>(_code.data()), _code.size()};
	//auto& cachedExec = Cache::registerExec(key, std::move(exec));
	Runtime runtime(_data, _env);
	auto mainFunc = (EntryFuncPtr)ee->getFunctionAddress(mainFuncName);
	auto returnCode = runEntryFunc(mainFunc, &runtime);
	if (returnCode == ReturnCode::Return)
		this->returnData = runtime.getReturnData();

	auto executionEndTime = std::chrono::high_resolution_clock::now();
	clog(JIT) << " + " << std::chrono::duration_cast<std::chrono::milliseconds>(executionEndTime - executionStartTime).count() << " ms ";
	//clog(JIT) << "Max stack size: " << Stack::maxStackSize;

	clog(JIT) << "\n";

	return returnCode;
}


}
}
}
