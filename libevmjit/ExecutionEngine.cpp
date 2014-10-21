
#include "ExecutionEngine.h"

#include <csetjmp>

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

#include <libevm/VM.h>

#include "Runtime.h"
#include "Memory.h"
#include "Type.h"

namespace dev
{
namespace eth
{
namespace jit
{

ExecutionEngine::ExecutionEngine()
{

}

extern "C" { EXPORT std::jmp_buf* rt_jmpBuf; }


int ExecutionEngine::run(std::unique_ptr<llvm::Module> _module, u256& _gas, ExtVMFace* _ext)
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
		BOOST_THROW_EXCEPTION(Exception() << errinfo_comment(errorMsg));
	_module.release();	// Successfully created llvm::ExecutionEngine takes ownership of the module
	exec->finalizeObject();

	// Create fake ExtVM interface
	if (!_ext)
	{
		_ext = new ExtVMFace;
		_ext->myAddress = Address(1122334455667788);
		_ext->caller = Address(0xfacefacefaceface);
		_ext->origin = Address(101010101010101010);
		_ext->value = 0xabcd;
		_ext->gasPrice = 1002;
		_ext->previousBlock.hash = u256(1003);
		_ext->currentBlock.coinbaseAddress = Address(1004);
		_ext->currentBlock.timestamp = 1005;
		_ext->currentBlock.number = 1006;
		_ext->currentBlock.difficulty = 1007;
		_ext->currentBlock.gasLimit = 1008;
		std::string calldata = "Hello the Beautiful World of Ethereum!";
		_ext->data = calldata;
		unsigned char fakecode[] = {0x0d, 0x0e, 0x0a, 0x0d, 0x0b, 0x0e, 0xe, 0xf};
		_ext->code = decltype(_ext->code)(fakecode, 8);
	}

	// Init runtime
	Runtime runtime(_gas, *_ext);

	auto entryFunc = module->getFunction("main");
	if (!entryFunc)
		BOOST_THROW_EXCEPTION(Exception() << errinfo_comment("main function not found"));

	ReturnCode returnCode;
	std::jmp_buf buf;
	auto r = setjmp(buf);
	if (r == 0)
	{
		rt_jmpBuf = &buf;
		auto result = exec->runFunction(entryFunc, {});
		returnCode = static_cast<ReturnCode>(result.IntVal.getZExtValue());
	}
	else
		returnCode = static_cast<ReturnCode>(r);

	// Return remaining gas
	_gas = returnCode == ReturnCode::OutOfGas ? 0 : Runtime::getGas();

	if (returnCode == ReturnCode::Return)
	{
		returnData = Memory::getReturnData().toVector(); // TODO: It might be better to place is in Runtime interface

		std::cout << "RETURN [ ";
		for (auto it = returnData.begin(), end = returnData.end(); it != end; ++it)
			std::cout << std::hex << std::setw(2) << std::setfill('0') << (int)*it << " ";
		std::cout << "]\n";
	}

	std::cout << "RETURN CODE: " << (int)returnCode << std::endl;
	return static_cast<int>(returnCode);
}

}
}
}
