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
struct ExecBundle
{
	std::unique_ptr<llvm::ExecutionEngine> engine;
	llvm::Function* entryFunc = nullptr;
};

class Cache
{
public:
	using Key = void const*;

	static ExecBundle& registerExec(Key _key, ExecBundle&& _exec);
	static ExecBundle* findExec(Key _key);
};

}
}
}
