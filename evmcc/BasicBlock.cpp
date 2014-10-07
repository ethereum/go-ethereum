
#include "BasicBlock.h"

#include <llvm/IR/Function.h>

namespace evmcc
{

const char* BasicBlock::NamePrefix = "Instr.";

BasicBlock::BasicBlock(ProgramCounter _beginInstIdx, llvm::Function* _mainFunc):
	m_beginInstIdx(_beginInstIdx),
	m_llvmBB(llvm::BasicBlock::Create(_mainFunc->getContext(), {NamePrefix, std::to_string(_beginInstIdx)}, _mainFunc))
{}

}