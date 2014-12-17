#pragma once

#include <memory>
#include <llvm/ExecutionEngine/ExecutionEngine.h>


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

class Cache
{
public:
	using Key = std::string;

	static ExecBundle& registerExec(Key _key, ExecBundle&& _exec);
	static ExecBundle* findExec(Key _key);
};

}
}
}
