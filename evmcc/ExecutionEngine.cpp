
#include "ExecutionEngine.h"

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

#include "Ext.h"

namespace evmcc
{

ExecutionEngine::ExecutionEngine()
{

}


int ExecutionEngine::run(std::unique_ptr<llvm::Module> _module)
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
		triple.setObjectFormat(llvm::Triple::ObjectFormatType::ELF);	// MCJIT does not support COFF format
	module->setTargetTriple(triple.str());

	auto exec = std::unique_ptr<llvm::ExecutionEngine>(builder.create());
	if (!exec)
	{
		if (!errorMsg.empty())
			std::cerr << "error creating EE: " << errorMsg << std::endl;
		else
			std::cerr << "unknown error creating llvm::ExecutionEngine" << std::endl;
		exit(1);
	}
	_module.release();	// Successfully created llvm::ExecutionEngine takes ownership of the module
	exec->finalizeObject();

	auto ext = std::make_unique<dev::eth::ExtVMFace>();
	ext->myAddress = dev::Address(1122334455667788);
	ext->caller = dev::Address(0xfacefacefaceface);
	ext->origin = dev::Address(101010101010101010);
	ext->value = 0xabcd;
	ext->gasPrice = 1002;
	std::string calldata = "Hello the Beautiful World of Ethereum!";
	ext->data = calldata;
	Ext::init(std::move(ext));

	auto entryFunc = module->getFunction("main");
	if (!entryFunc)
	{
		std::cerr << "main function not found\n";
		exit(1);
	}

	auto result = exec->runFunction(entryFunc, {});
	if (auto intResult = result.IntVal.getZExtValue())
	{
		auto index = intResult >> 32;
		auto size = 0xFFFFFFFF & intResult;
		// TODO: Get the data from memory
		return 10;
	}	
	return 0;
}

}