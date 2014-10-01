#pragma once

#include <llvm/IR/IRBuilder.h>

#include <libdevcore/Common.h>

namespace evmcc
{

class Memory
{
public:
	Memory(llvm::IRBuilder<>& _builder);

	static const dev::bytes& init();

	llvm::Value* loadWord(llvm::Value* _addr);
	void storeWord(llvm::Value* _addr, llvm::Value* _word);
	void storeByte(llvm::Value* _addr, llvm::Value* _byte);
	llvm::Value* getSize();

	void dump(uint64_t _begin, uint64_t _end = 0);

private:
	llvm::IRBuilder<>& m_builder;

	llvm::Function* m_memRequire;
	llvm::Function* m_memDump;
	llvm::Function* m_memSize;
};

}
