#pragma once

#include <llvm/IR/IRBuilder.h>

namespace evmcc
{

class Memory
{
public:
	Memory(llvm::IRBuilder<>& _builder, llvm::Module* _module);

	llvm::Value* loadByte(llvm::Value* _addr);
	void storeWord(llvm::Value* _addr, llvm::Value* _word);
	void storeByte(llvm::Value* _addr, llvm::Value* _byte);

	void dump(uint64_t _begin, uint64_t _end);

private:
	llvm::IRBuilder<>& m_builder;

	llvm::Function* m_memRequire;
	llvm::Function* m_memDump;
};

}
