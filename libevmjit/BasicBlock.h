
#include <vector>

#include <llvm/IR/BasicBlock.h>

namespace dev
{
namespace eth
{
namespace jit
{

using ProgramCounter = uint64_t; // TODO: Rename
	
class BasicBlock
{
public:
	class LocalStack
	{
	public:
		/// Pushes value on stack
		void push(llvm::Value* _value);

		/// Pops and returns top value
		llvm::Value* pop();

		/// Gets _index'th value from top (counting from 0)
		llvm::Value* get(size_t _index);

		/// Duplicates _index'th value on stack.
		void dup(size_t _index);

		/// Swaps _index'th value on stack with a value on stack top.
		/// @param _index Index of value to be swaped. Cannot be 0.
		void swap(size_t _index);

		/// Size of the stack
		size_t size() const { return m_backend.size(); }

		size_t initialSize() const { return m_numRequiredStackItems; }

	private:
		// LocalStack(llvm::BasicBlock* _llvmBB) : m_llvmBB(_llvmBB) {}
		LocalStack(llvm::BasicBlock* _llvmBB) : m_llvmBB(_llvmBB), m_numRequiredStackItems(0) {}
		LocalStack(LocalStack const&) = delete;
		void operator=(LocalStack const&) = delete;
		friend BasicBlock;

	private:
		std::vector<llvm::Value*> m_backend;

		/** Basic block into which phi nodes are inserted */
		llvm::BasicBlock* m_llvmBB;

		/** Number of items required on the EVM stack at the beginning of the block */
		size_t m_numRequiredStackItems;
	};

	/// Basic block name prefix. The rest is beging instruction index.
	static const char* NamePrefix;

	explicit BasicBlock(ProgramCounter _beginInstIdx, ProgramCounter _endInstIdx, llvm::Function* _mainFunc);
	explicit BasicBlock(std::string _name, llvm::Function* _mainFunc);

	BasicBlock(const BasicBlock&) = delete;
	void operator=(const BasicBlock&) = delete;

	operator llvm::BasicBlock*() { return m_llvmBB; }
	llvm::BasicBlock* llvm() { return m_llvmBB; }

	LocalStack& getStack() { return m_stack; }

	ProgramCounter begin() { return m_beginInstIdx; }
	ProgramCounter end() { return m_endInstIdx; }

private:
	ProgramCounter const m_beginInstIdx;
	ProgramCounter const m_endInstIdx;
	llvm::BasicBlock* const m_llvmBB;

	/// Basic black state vector (stack) - current/end values and their positions on stack
	/// @internal Must be AFTER m_llvmBB
	LocalStack m_stack;
};

}
}
}

