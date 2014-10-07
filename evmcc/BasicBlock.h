
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

	explicit BasicBlock(ProgramCounter _beginInstIdx, ProgramCounter _endInstIdx, llvm::Function* _mainFunc);

	BasicBlock(const BasicBlock&) = delete;
	void operator=(const BasicBlock&) = delete;

	operator llvm::BasicBlock*() { return m_llvmBB; }
	llvm::BasicBlock* llvm() { return m_llvmBB; }

	State& getState() { return m_state; }

	void setEnd(ProgramCounter _endInstIdx) { m_endInstIdx = _endInstIdx; }

private:
	ProgramCounter m_beginInstIdx;
	ProgramCounter m_endInstIdx;
	llvm::BasicBlock* m_llvmBB;

	/// Basic black state vector - current/end values and their positions
	State m_state;
};

}