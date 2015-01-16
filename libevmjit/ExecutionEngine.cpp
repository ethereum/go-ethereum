#include "ExecutionEngine.h"

#include <chrono>

#include <llvm/IR/Module.h>
#include <llvm/ADT/Triple.h>
#pragma warning(push)
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wunused-parameter"
#include <llvm/ExecutionEngine/ExecutionEngine.h>
#include <llvm/ExecutionEngine/SectionMemoryManager.h>
#pragma warning(pop)
#pragma GCC diagnostic pop
#include <llvm/ExecutionEngine/MCJIT.h>
#include <llvm/Support/TargetSelect.h>
#include <llvm/Support/Host.h>

#include "Runtime.h"
#include "Compiler.h"
#include "Cache.h"

#if defined(NDEBUG)
#define DEBUG_ENV_OPTION(name) false
#else
#include <cstdlib>
#define DEBUG_ENV_OPTION(name) (std::getenv(#name) != nullptr)
#endif


extern "C" void env_sha3(dev::eth::jit::byte const* _begin, uint64_t _size, std::array<dev::eth::jit::byte, 32>* o_hash);

namespace dev
{
namespace eth
{
namespace jit
{

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

std::string codeHash(bytes const& _code)
{
	std::array<byte, 32> binHash;
	env_sha3(_code.data(), _code.size(), &binHash);

	std::ostringstream os;
	for (auto i: binHash)
		os << std::hex << std::setfill('0') << std::setw(2) << (int)(std::make_unsigned<decltype(i)>::type)i;

	return os.str();
}

}

ReturnCode ExecutionEngine::run(bytes const& _code, RuntimeData* _data, Env* _env)
{
	static std::unique_ptr<llvm::ExecutionEngine> ee;  // TODO: Use Managed Objects from LLVM?
	static auto debugDumpModule = DEBUG_ENV_OPTION(EVMJIT_DUMP_MODULE);
	static bool objectCacheEnabled = !DEBUG_ENV_OPTION(EVMJIT_CACHE_OFF);

	auto mainFuncName = codeHash(_code);
	EntryFuncPtr entryFuncPtr{};
	Runtime runtime(_data, _env);	// TODO: I don't know why but it must be created before getFunctionAddress() calls

	if (ee && (entryFuncPtr = (EntryFuncPtr)ee->getFunctionAddress(mainFuncName)))
	{
	}
	else
	{
		auto objectCache = objectCacheEnabled ? Cache::getObjectCache() : nullptr;
		std::unique_ptr<llvm::Module> module;
		if (objectCache)
			module = Cache::getObject(mainFuncName);
		if (!module)
			module = Compiler({}).compile(_code, mainFuncName);
		if (debugDumpModule)
			module->dump();
		if (!ee)
		{
			llvm::InitializeNativeTarget();
			llvm::InitializeNativeTargetAsmPrinter();

			llvm::EngineBuilder builder(module.get());
			builder.setEngineKind(llvm::EngineKind::JIT);
			builder.setUseMCJIT(true);
			std::unique_ptr<llvm::SectionMemoryManager> memoryManager(new llvm::SectionMemoryManager);
			builder.setMCJITMemoryManager(memoryManager.get());
			builder.setOptLevel(llvm::CodeGenOpt::None);

			auto triple = llvm::Triple(llvm::sys::getProcessTriple());
			if (triple.getOS() == llvm::Triple::OSType::Win32)
				triple.setObjectFormat(llvm::Triple::ObjectFormatType::ELF);  // MCJIT does not support COFF format
			module->setTargetTriple(triple.str());

			ee.reset(builder.create());
			if (!ee)
				return ReturnCode::LLVMConfigError;

			module.release();         // Successfully created llvm::ExecutionEngine takes ownership of the module
			memoryManager.release();  // and memory manager

			if (objectCache)
				ee->setObjectCache(objectCache);
			entryFuncPtr = (EntryFuncPtr)ee->getFunctionAddress(mainFuncName);
		}
		else
		{
			if (!entryFuncPtr)
			{
				ee->addModule(module.get());
				module.release();
				entryFuncPtr = (EntryFuncPtr)ee->getFunctionAddress(mainFuncName);
			}
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
