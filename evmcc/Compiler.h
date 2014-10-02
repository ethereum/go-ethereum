
#pragma once

#include <llvm/IR/Module.h>

#include <libdevcore/Common.h>

namespace evmcc
{

class Compiler
{
public:

	using ProgramCounter = uint64_t;

	Compiler();

	std::unique_ptr<llvm::Module> compile(const dev::bytes& bytecode);

private:

	llvm::BasicBlock* getOrCreateBasicBlockAtPC(ProgramCounter pc);
	void createBasicBlocks(const dev::bytes& bytecode);

	/**
	 *  Maps a program counter pc to a basic block which starts at pc (if any).
	 */
	std::map<ProgramCounter, llvm::BasicBlock*> basicBlocks;

	/**
	 *  Maps a pc at which there is a JUMP or JUMPI to the target block of the jump.
	 */
	std::map<ProgramCounter, llvm::BasicBlock*> jumpTargets;
};

}
