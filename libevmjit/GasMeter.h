
#pragma once

#include <libevmcore/Instruction.h>

#include "CompilerHelper.h"

namespace dev
{
namespace eth
{
namespace jit
{
class RuntimeManager;

class GasMeter : public CompilerHelper // TODO: Use RuntimeHelper
{
public:
	GasMeter(llvm::IRBuilder<>& _builder, RuntimeManager& _runtimeManager);

	/// Count step cost of instruction
	void count(Instruction _inst);

	/// Calculate & count gas cost for SSTORE instruction
	void countSStore(class Ext& _ext, llvm::Value* _index, llvm::Value* _newValue);

	/// Finalize cost-block by checking gas needed for the block before the block
	/// @param _additionalCost adds additional cost to cost-block before commit
	void commitCostBlock(llvm::Value* _additionalCost = nullptr);

	/// Give back an amount of gas not used by a call
	void giveBack(llvm::Value* _gas);

	/// Generate code that checks the cost of additional memory used by program
	void checkMemory(llvm::Value* _additionalMemoryInWords);

private:
	/// Cumulative gas cost of a block of instructions
	/// @TODO Handle overflow
	uint64_t m_blockCost = 0;

	llvm::CallInst* m_checkCall = nullptr;
	llvm::Function* m_gasCheckFunc = nullptr;

	RuntimeManager& m_runtimeManager;
};

}
}
}

