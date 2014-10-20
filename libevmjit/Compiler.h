
#pragma once

#include <llvm/IR/IRBuilder.h>

#include <libdevcore/Common.h>

#include "BasicBlock.h"

namespace dev
{
namespace eth
{
namespace jit
{

class Compiler
{
public:

	using ProgramCounter = uint64_t;

	Compiler();

	std::unique_ptr<llvm::Module> compile(bytesConstRef bytecode);

	void dumpBasicBlockGraph(std::ostream& out);


private:

	void createBasicBlocks(bytesConstRef bytecode);

	void compileBasicBlock(BasicBlock& basicBlock, bytesConstRef bytecode, class Memory& memory, class Ext& ext, class GasMeter& gasMeter, llvm::BasicBlock* nextBasicBlock);

	void linkBasicBlocks();


	llvm::IRBuilder<> m_builder;

	/**
	 *  Maps a program counter pc to a basic block that starts at pc (if any).
	 */
	std::map<ProgramCounter, BasicBlock> basicBlocks;

	/**
	 *  Maps a pc at which there is a JUMP or JUMPI to the target block of the jump.
	 */
	std::map<ProgramCounter, llvm::BasicBlock*> m_directJumpTargets;

	/**
	 *  A list of possible blocks to which there may be indirect jumps.
	 */
	std::vector<BasicBlock*> m_indirectJumpTargets;

	/// Stop basic block - terminates execution with STOP code (0)
	llvm::BasicBlock* m_stopBB = nullptr;

	/**
	 *  Block with a jump table.
	 */
	std::unique_ptr<BasicBlock> m_jumpTableBlock;

	/**
	 *  Default destination for indirect jumps.
	 */
	std::unique_ptr<BasicBlock> m_badJumpBlock;

	/// Main program function
	llvm::Function* m_mainFunc = nullptr;
};

}
}
}
