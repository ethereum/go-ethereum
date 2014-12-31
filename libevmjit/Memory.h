#pragma once

#include "CompilerHelper.h"

namespace dev
{
namespace eth
{
namespace jit
{
class GasMeter;

class Memory : public RuntimeHelper
{
public:
	Memory(RuntimeManager& _runtimeManager, GasMeter& _gasMeter);

	llvm::Value* loadWord(llvm::Value* _addr);
	void storeWord(llvm::Value* _addr, llvm::Value* _word);
	void storeByte(llvm::Value* _addr, llvm::Value* _byte);
	llvm::Value* getData();
	llvm::Value* getSize();
	llvm::Value* getBytePtr(llvm::Value* _index);
	void copyBytes(llvm::Value* _srcPtr, llvm::Value* _srcSize, llvm::Value* _srcIndex,
				   llvm::Value* _destMemIdx, llvm::Value* _byteCount);

	/// Requires the amount of memory to for data defined by offset and size. And counts gas fee for that memory.
	void require(llvm::Value* _offset, llvm::Value* _size);

private:
	GasMeter& m_gasMeter;

	llvm::Function* createFunc(bool _isStore, llvm::Type* _type, GasMeter& _gasMeter);
	llvm::Function* createRequireFunc(GasMeter& _gasMeter);

	llvm::Function* m_resize;
	llvm::Function* m_require;
	llvm::Function* m_loadWord;
	llvm::Function* m_storeWord;
	llvm::Function* m_storeByte;
};

}
}
}

