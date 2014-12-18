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
namespace
{
	using CacheMap = std::unordered_map<Cache::Key, ExecBundle>;

	CacheMap& getCacheMap()
	{
		static CacheMap map;
		return map;
	}
}

//#define LOG(...) std::cerr << "CACHE "
#define LOG(...) std::ostream(nullptr)

ExecBundle& Cache::registerExec(Cache::Key _key, ExecBundle&& _exec)
{
	auto& map = getCacheMap();
	auto r = map.insert(std::make_pair(_key, std::move(_exec)));
	assert(r.second && "Updating cached objects not supported");
	LOG() << "add\n";
	return r.first->second;  // return exec, now owned by cache
}

ExecBundle* Cache::findExec(Cache::Key _key)
{
	auto& map = getCacheMap();
	auto it = map.find(_key);
	if (it != map.end())
	{
		LOG() << "hit\n";
		return &it->second;
	}
	LOG() << "miss\n";
	return nullptr;
}

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
