#include "Cache.h"
#include <unordered_map>
#include <cassert>
#include <iostream>

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

}
}
}
