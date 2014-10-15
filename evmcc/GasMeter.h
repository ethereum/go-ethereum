
#pragma once

#include <llvm/IR/IRBuilder.h>

#include <libevmface/Instruction.h>

namespace dev
{
namespace eth
{
namespace jit
{

class GasMeter
{
public:
	GasMeter(llvm::IRBuilder<>& _builder, llvm::Module* _module);

	GasMeter(const GasMeter&) = delete;
	void operator=(GasMeter) = delete;

	/// Count step cost of instruction
	void count(dev::eth::Instruction _inst);

	/// Calculate & count gas cost for SSTORE instruction
	void countSStore(class Ext& _ext, llvm::Value* _index, llvm::Value* _newValue);

	/// Finalize cost-block by checking gas needed for the block before the block
	/// @param _additionalCost adds additional cost to cost-block before commit
	void commitCostBlock(llvm::Value* _additionalCost = nullptr);

	/// Give back an amount of gas not used by a call
	void giveBack(llvm::Value* _gas);

	/// Generate code that checks the cost of additional memory used by program
	void checkMemory(llvm::Value* _additionalMemoryInWords, llvm::IRBuilder<>& _builder);

	llvm::Value* getGas();

private:
	/// Cumulative gas cost of a block of instructions
	/// @TODO Handle overflow
	uint64_t m_blockCost = 0;
	llvm::IRBuilder<>& m_builder;
	llvm::CallInst* m_checkCall = nullptr;
	llvm::GlobalVariable* m_gas;
	llvm::Function* m_gasCheckFunc;
};

}
}
}

