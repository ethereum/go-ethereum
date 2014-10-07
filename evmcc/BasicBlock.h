
#include <llvm/IR/BasicBlock.h>

namespace evmcc
{

using ProgramCounter = uint64_t;
	
class BasicBlock
{
public:
	explicit BasicBlock(ProgramCounter _instIdx, llvm::Function* _mainFunc);
	BasicBlock(const BasicBlock&) = delete;
	void operator=(const BasicBlock&) = delete;

	operator llvm::BasicBlock*() { return m_llvmBB; }

private:
	llvm::BasicBlock* m_llvmBB;
};

}