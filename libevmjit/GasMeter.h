
#pragma once

#include "CompilerHelper.h"
#include "Instruction.h"

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

	/// Count additional cost
	void count(llvm::Value* _cost);

	/// Calculate & count gas cost for SSTORE instruction
	void countSStore(class Ext& _ext, llvm::Value* _index, llvm::Value* _newValue);

	/// Calculate & count additional gas cost for EXP instruction
	void countExp(llvm::Value* _exponent);

	/// Count gas cost of LOG data
	void countLogData(llvm::Value* _dataLength);

	/// Count gas cost of SHA3 data
	void countSha3Data(llvm::Value* _dataLength);

	/// Finalize cost-block by checking gas needed for the block before the block
	void commitCostBlock();

	/// Give back an amount of gas not used by a call
	void giveBack(llvm::Value* _gas);

	/// Generate code that checks the cost of additional memory used by program
	void countMemory(llvm::Value* _additionalMemoryInWords);

	/// Count addional gas cost for memory copy
	void countCopy(llvm::Value* _copyWords);

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

