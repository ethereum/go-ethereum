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
	if (auto cachedExec = Cache::findExec(key))
	{
		return run(*cachedExec, _data, _env);
	}

	auto module = Compiler({}).compile(_code);
	return run(std::move(module), _data, _env, _code);
}

ReturnCode ExecutionEngine::run(std::unique_ptr<llvm::Module> _module, RuntimeData* _data, Env* _env, bytes const& _code)
{
	auto module = _module.get(); // Keep ownership of the module in _module

	llvm::sys::PrintStackTraceOnErrorSignal();
	static const auto program = "EVM JIT";
	llvm::PrettyStackTraceProgram X(1, &program);

	auto&& context = llvm::getGlobalContext();

	llvm::InitializeNativeTarget();
	llvm::InitializeNativeTargetAsmPrinter();
	llvm::InitializeNativeTargetAsmParser();

	std::string errorMsg;
	llvm::EngineBuilder builder(module);
	//builder.setMArch(MArch);
	//builder.setMCPU(MCPU);
	//builder.setMAttrs(MAttrs);
	//builder.setRelocationModel(RelocModel);
	//builder.setCodeModel(CMModel);
	builder.setErrorStr(&errorMsg);
	builder.setEngineKind(llvm::EngineKind::JIT);
	builder.setUseMCJIT(true);
	builder.setMCJITMemoryManager(new llvm::SectionMemoryManager());
	builder.setOptLevel(llvm::CodeGenOpt::None);

	auto triple = llvm::Triple(llvm::sys::getProcessTriple());
	if (triple.getOS() == llvm::Triple::OSType::Win32)
		triple.setObjectFormat(llvm::Triple::ObjectFormatType::ELF);    // MCJIT does not support COFF format
	module->setTargetTriple(triple.str());

	ExecBundle exec;
	exec.engine.reset(builder.create());
	if (!exec.engine)
		return ReturnCode::LLVMConfigError;
	_module.release();  // Successfully created llvm::ExecutionEngine takes ownership of the module

	auto finalizationStartTime = std::chrono::high_resolution_clock::now();
	exec.engine->finalizeObject();
	auto finalizationEndTime = std::chrono::high_resolution_clock::now();
	clog(JIT) << " + " << std::chrono::duration_cast<std::chrono::milliseconds>(finalizationEndTime - finalizationStartTime).count();

	auto executionStartTime = std::chrono::high_resolution_clock::now();

	exec.entryFunc = module->getFunction("main");
	if (!exec.entryFunc)
		return ReturnCode::LLVMLinkError;

	std::string key{reinterpret_cast<char const*>(_code.data()), _code.size()};
	auto& cachedExec = Cache::registerExec(key, std::move(exec));
	auto returnCode = run(cachedExec, _data, _env);

	auto executionEndTime = std::chrono::high_resolution_clock::now();
	clog(JIT) << " + " << std::chrono::duration_cast<std::chrono::milliseconds>(executionEndTime - executionStartTime).count() << " ms ";
	//clog(JIT) << "Max stack size: " << Stack::maxStackSize;

	clog(JIT) << "\n";

	return returnCode;
}

ReturnCode ExecutionEngine::run(ExecBundle const& _exec, RuntimeData* _data, Env* _env)
{
	ReturnCode returnCode;
	Runtime runtime(_data, _env);


	std::vector<llvm::GenericValue> args{{}, llvm::GenericValue(&runtime)};
	llvm::GenericValue result;

	typedef ReturnCode(*EntryFuncPtr)(int, Runtime*);

	auto entryFuncVoidPtr = _exec.engine->getPointerToFunction(_exec.entryFunc);
	auto entryFuncPtr = static_cast<EntryFuncPtr>(entryFuncVoidPtr);

	auto r = setjmp(runtime.getJmpBuf());
	if (r == 0)
	{
		returnCode = entryFuncPtr(0, &runtime);
	}
	else
		returnCode = static_cast<ReturnCode>(r);

	if (returnCode == ReturnCode::Return)
	{
		returnData = runtime.getReturnData();

		auto&& log = clog(JIT);
		log << "RETURN [ ";
		for (auto it = returnData.begin(), end = returnData.end(); it != end; ++it)
			log << std::hex << std::setw(2) << std::setfill('0') << (int)*it << " ";
		log << "]";
	}
	else
		clog(JIT) << "RETURN " << (int)returnCode;

	return returnCode;
}

}
}
}
