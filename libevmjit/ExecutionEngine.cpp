#include "ExecutionEngine.h"

#include <csetjmp>
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

namespace dev
{
namespace eth
{
namespace jit
{

int ExecutionEngine::run(std::unique_ptr<llvm::Module> _module, RuntimeData* _data, Env* _env)
{
	auto module = _module.get(); // Keep ownership of the module in _module

	llvm::sys::PrintStackTraceOnErrorSignal();
	static const auto program = "evmcc";
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

	auto exec = std::unique_ptr<llvm::ExecutionEngine>(builder.create());
	if (!exec)
		return -1; // FIXME: Handle internal errors
	_module.release();  // Successfully created llvm::ExecutionEngine takes ownership of the module

	auto finalizationStartTime = std::chrono::high_resolution_clock::now(); // FIXME: It's not compilation time
	exec->finalizeObject();
	auto finalizationEndTime = std::chrono::high_resolution_clock::now();
	clog(JIT) << " + " << std::chrono::duration_cast<std::chrono::milliseconds>(finalizationEndTime - finalizationStartTime).count();

	auto executionStartTime = std::chrono::high_resolution_clock::now();

	auto entryFunc = module->getFunction("main");
	if (!entryFunc)
		return -2; // FIXME: Handle internal errors

	ReturnCode returnCode;
	std::jmp_buf buf;
	Runtime runtime(_data, _env, buf);
	auto r = setjmp(buf);
	if (r == 0)
	{
		auto result = exec->runFunction(entryFunc, {{}, llvm::GenericValue(&runtime)});
		returnCode = static_cast<ReturnCode>(result.IntVal.getZExtValue());
	}
	else
		returnCode = static_cast<ReturnCode>(r);
	
	auto executionEndTime = std::chrono::high_resolution_clock::now();
	clog(JIT) << " + " << std::chrono::duration_cast<std::chrono::milliseconds>(executionEndTime - executionStartTime).count() << " ms ";
	//clog(JIT) << "Max stack size: " << Stack::maxStackSize;

	if (returnCode == ReturnCode::Return)
	{
		returnData = runtime.getReturnData(); // TODO: It might be better to place is in Runtime interface

		auto&& log = clog(JIT);
		log << "RETURN [ ";
		for (auto it = returnData.begin(), end = returnData.end(); it != end; ++it)
			log << std::hex << std::setw(2) << std::setfill('0') << (int)*it << " ";
		log << "]";
	}
	else
		clog(JIT) << "RETURN " << (int)returnCode;

	clog(JIT) << "\n";

	return static_cast<int>(returnCode);
}

}
}
}
