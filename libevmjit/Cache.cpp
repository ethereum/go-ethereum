#include "Cache.h"
#include <unordered_map>
#include <cassert>
#include <iostream>
#include <llvm/IR/Module.h>

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


void ObjectCache::notifyObjectCompiled(llvm::Module const* _module, llvm::MemoryBuffer const* _object)
{
	auto&& key = _module->getModuleIdentifier();
	std::unique_ptr<llvm::MemoryBuffer> obj(llvm::MemoryBuffer::getMemBufferCopy(_object->getBuffer()));
	m_map.insert(std::make_pair(key, std::move(obj)));
}

llvm::MemoryBuffer* ObjectCache::getObject(llvm::Module const* _module)
{
	auto it = m_map.find(_module->getModuleIdentifier());
	if (it != m_map.end())
		return llvm::MemoryBuffer::getMemBufferCopy(it->second->getBuffer());
	return nullptr;
}

}
}
}
