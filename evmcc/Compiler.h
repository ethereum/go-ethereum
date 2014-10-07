
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

	BasicBlock& getOrCreateBasicBlockAtPC(ProgramCounter pc);
	void createBasicBlocks(const dev::bytes& bytecode);

	void linkBasicBlocks();

	/**
	 *  Maps a program counter pc to a basic block which starts at pc (if any).
	 */
	std::map<ProgramCounter, BasicBlock> basicBlocks;

	/**
	 *  Maps a pc at which there is a JUMP or JUMPI to the target block of the jump.
	 */
	std::map<ProgramCounter, llvm::BasicBlock*> jumpTargets;

private:
	/// Collection of basic blocks in program
	//std::vector<BasicBlock> m_basicBlocks;

	/// Main program function
	llvm::Function* m_mainFunc;
};

}
