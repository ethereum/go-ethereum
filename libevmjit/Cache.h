#pragma once

#include <memory>
#include <unordered_map>
#include <llvm/ExecutionEngine/ExecutionEngine.h>
#include <llvm/ExecutionEngine/ObjectCache.h>


namespace dev
{
namespace eth
{
namespace jit
{

/// A bundle of objects and information needed for a contract execution
class ExecBundle
{
public:
	std::unique_ptr<llvm::ExecutionEngine> engine;

	ExecBundle() = default;
	ExecBundle(ExecBundle&& _other):
		engine(std::move(_other.engine))
	{}

	ExecBundle(ExecBundle const&) = delete;
	void operator=(ExecBundle) = delete;
};


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
	using Key = std::string;

	static ExecBundle& registerExec(Key _key, ExecBundle&& _exec);
	static ExecBundle* findExec(Key _key);

	static ObjectCache* getObjectCache();
};

}
}
}
