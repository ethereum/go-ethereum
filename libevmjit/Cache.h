#pragma once

#include <memory>
#include <unordered_map>
#include <llvm/ExecutionEngine/ObjectCache.h>


namespace dev
{
namespace eth
{
namespace jit
{

class ObjectCache : public llvm::ObjectCache
{
public:
	/// notifyObjectCompiled - Provides a pointer to compiled code for Module M.
	virtual void notifyObjectCompiled(llvm::Module const* _module, llvm::MemoryBuffer const* _object) final override;

	/// getObjectCopy - Returns a pointer to a newly allocated MemoryBuffer that
	/// contains the object which corresponds with Module M, or 0 if an object is
	/// not available. The caller owns both the MemoryBuffer returned by this
	/// and the memory it references.
	virtual llvm::MemoryBuffer* getObject(llvm::Module const* _module) final override;

private:
	std::unordered_map<std::string, std::unique_ptr<llvm::MemoryBuffer>> m_map;
};


class Cache
{
public:
	static ObjectCache* getObjectCache();
	static std::unique_ptr<llvm::Module> getObject(std::string const& id);
};

}
}
}
