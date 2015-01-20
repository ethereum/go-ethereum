#include "Cache.h"
#include <unordered_map>
#include <cassert>
#include <iostream>
#include <llvm/IR/Module.h>
#include <llvm/IR/LLVMContext.h>
#include <llvm/Support/Path.h>
#include <llvm/Support/FileSystem.h>
#include <llvm/Support/raw_os_ostream.h>

namespace dev
{
namespace eth
{
namespace jit
{

//#define LOG(...) std::cerr << "CACHE "
#define LOG(...) std::ostream(nullptr)

ObjectCache* Cache::getObjectCache()
{
	static ObjectCache objectCache;
	return &objectCache;
}

namespace
{
	llvm::MemoryBuffer* lastObject;
}

std::unique_ptr<llvm::Module> Cache::getObject(std::string const& id)
{
	assert(!lastObject);
	llvm::SmallString<256> cachePath;
	llvm::sys::path::system_temp_directory(false, cachePath);
	llvm::sys::path::append(cachePath, "evm_objs", id);

#if defined(__GNUC__) && !defined(NDEBUG)
	llvm::sys::fs::file_status st;
	auto err = llvm::sys::fs::status(cachePath.str(), st);
	if (err)
		return nullptr;
	auto mtime = st.getLastModificationTime().toEpochTime();

	std::tm tm;
	strptime(__DATE__ __TIME__, " %b %d %Y %H:%M:%S", &tm);
	auto btime = (uint64_t)std::mktime(&tm);
	if (btime > mtime)
		return nullptr;
#endif

	if (auto r = llvm::MemoryBuffer::getFile(cachePath.str(), -1, false))
		lastObject = llvm::MemoryBuffer::getMemBufferCopy(r.get()->getBuffer());
	else if (r.getError() != std::make_error_code(std::errc::no_such_file_or_directory))
		std::cerr << r.getError().message(); // TODO: Add log

	if (lastObject)  // if object found create fake module
	{
		auto module = std::unique_ptr<llvm::Module>(new llvm::Module(id, llvm::getGlobalContext()));
		auto mainFuncType = llvm::FunctionType::get(llvm::IntegerType::get(llvm::getGlobalContext(), 32), {}, false);
		llvm::Function::Create(mainFuncType, llvm::Function::ExternalLinkage, id, module.get());
	}
	return nullptr;
}


void ObjectCache::notifyObjectCompiled(llvm::Module const* _module, llvm::MemoryBuffer const* _object)
{
	auto&& id = _module->getModuleIdentifier();
	llvm::SmallString<256> cachePath;
	llvm::sys::path::system_temp_directory(false, cachePath);
	llvm::sys::path::append(cachePath, "evm_objs");

	if (llvm::sys::fs::create_directory(cachePath.str()))
		return; // TODO: Add log

	llvm::sys::path::append(cachePath, id);

	std::string error;
	llvm::raw_fd_ostream cacheFile(cachePath.c_str(), error, llvm::sys::fs::F_None);
	cacheFile << _object->getBuffer();
}

llvm::MemoryBuffer* ObjectCache::getObject(llvm::Module const*)
{
	auto o = lastObject;
	lastObject = nullptr;
	return o;
}

}
}
}
