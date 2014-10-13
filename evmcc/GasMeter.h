
#pragma once

#include <llvm/IR/IRBuilder.h>

#include <libevmface/Instruction.h>

namespace evmcc
{

class GasMeter
{
public:
	GasMeter(llvm::IRBuilder<>& _builder, llvm::Module* module);

	GasMeter(const GasMeter&) = delete;
	void operator=(GasMeter) = delete;

	/// Count step cost of instruction
	void count(dev::eth::Instruction _inst);

	/// Finalize cost block by checking gas needed for the block before the block
	void commitCostBlock();

	/// Generate code that checks the cost of additional memory used by program
	void checkMemory(llvm::Value* _additionalMemoryInWords, llvm::IRBuilder<>& _builder);

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
