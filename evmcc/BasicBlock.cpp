
#include "BasicBlock.h"

#include <llvm/IR/Function.h>

namespace evmcc
{

BasicBlock::BasicBlock(ProgramCounter _instIdx, llvm::Function* _mainFunc)
	: m_llvmBB(llvm::BasicBlock::Create(_mainFunc->getContext(), {"Instr.", std::to_string(_instIdx)}, _mainFunc))
{}

}