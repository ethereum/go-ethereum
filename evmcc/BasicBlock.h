
#include <vector>

#include <llvm/IR/BasicBlock.h>

namespace evmcc
{

using ProgramCounter = uint64_t;
	
class BasicBlock
{
public:
	using State = std::vector<llvm::Value*>;

	static const char* NamePrefix;

	explicit BasicBlock(ProgramCounter _beginInstIdx, llvm::Function* _mainFunc);

	BasicBlock(const BasicBlock&) = delete;
	void operator=(const BasicBlock&) = delete;

	operator llvm::BasicBlock*() { return m_llvmBB; }
	llvm::BasicBlock* llvm() { return m_llvmBB; }

	State& getState() { return m_state; }

private:
	ProgramCounter m_beginInstIdx;
	llvm::BasicBlock* m_llvmBB;

	/// Basic black state vector - current/end values and their positions
	State m_state;
};

}