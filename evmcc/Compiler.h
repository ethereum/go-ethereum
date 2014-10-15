
#pragma once

#include <llvm/IR/Module.h>

#include <libdevcore/Common.h>

#include "BasicBlock.h"

namespace evmcc
{

class Compiler
{
public:

	using ProgramCounter = uint64_t;

	Compiler();

	std::unique_ptr<llvm::Module> compile(const dev::bytes& bytecode);

private:

	void createBasicBlocks(const dev::bytes& bytecode);

	void linkBasicBlocks();

	/**
	 *  Maps a program counter pc to a basic block which starts at pc (if any).
	 */
	std::map<ProgramCounter, BasicBlock> basicBlocks;

	/**
	 *  Maps a pc at which there is a JUMP or JUMPI to the target block of the jump.
	 */
	std::map<ProgramCounter, BasicBlock*> m_directJumpTargets;

	/**
	 *  A list of possible blocks to which there may be indirect jumps.
	 */
	std::vector<BasicBlock*> m_indirectJumpTargets;

	/// Collection of basic blocks in program
	//std::vector<BasicBlock> m_basicBlocks;

	/**
	 *  Final block for normal (non-exceptional) execution.
	 */
	std::unique_ptr<BasicBlock> m_finalBlock;

	/**
	 *  Block with a jump table.
	 */
	std::unique_ptr<BasicBlock> m_jumpTableBlock;

	/**
	 *  Default destination for indirect jumps.
	 */
	std::unique_ptr<BasicBlock> m_badJumpBlock;

	std::unique_ptr<BasicBlock> m_outOfGasBlock;

	/// Main program function
	llvm::Function* m_mainFunc = nullptr;
};

}
