#pragma once

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

};

}
}
}
