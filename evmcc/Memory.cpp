#include "Memory.h"

#include <vector>
#include <iostream>
#include <iomanip>
#include <cstdint>
#include <cassert>

#include <llvm/IR/Function.h>

#include <libdevcore/Common.h>

#include "Utils.h"

#ifdef _MSC_VER
	#define EXPORT __declspec(dllexport)
#else
	#define EXPORT
#endif

namespace evmcc
{

using MemoryImpl = dev::bytes;

static MemoryImpl* evmccrt_memory;


Memory::Memory(llvm::IRBuilder<>& _builder)
	: m_builder(_builder)
{
	auto voidTy	= m_builder.getVoidTy();
	auto i64Ty = m_builder.getInt64Ty();
	auto module = _builder.GetInsertBlock()->getParent()->getParent();


	auto memRequireTy = llvm::FunctionType::get(m_builder.getInt8PtrTy(), i64Ty, false);
	m_memRequire = llvm::Function::Create(memRequireTy,
	                                      llvm::GlobalValue::LinkageTypes::ExternalLinkage,
	                                      "evmccrt_memory_require", module);

	auto memSizeTy = llvm::FunctionType::get(i64Ty, false);
	m_memSize = llvm::Function::Create(memSizeTy,
	                                   llvm::GlobalValue::LinkageTypes::ExternalLinkage,
	                                   "evmccrt_memory_size", module);

	std::vector<llvm::Type*> argTypes = {i64Ty, i64Ty};
	auto dumpTy = llvm::FunctionType::get(m_builder.getVoidTy(), llvm::ArrayRef<llvm::Type*>(argTypes), false);
 	m_memDump = llvm::Function::Create(dumpTy, llvm::GlobalValue::LinkageTypes::ExternalLinkage,
 	                                   "evmccrt_memory_dump", module);
}

const dev::bytes& Memory::init()
{
	evmccrt_memory = new MemoryImpl();
	std::cerr << "MEMORY: create(), initial size = " << evmccrt_memory->size()
		<< std::endl;

	return *evmccrt_memory;
}


llvm::Value* Memory::loadWord(llvm::Value* _addr)
{
	// trunc _addr (an i256) to i64 index and use it to index the memory
	auto index = m_builder.CreateTrunc(_addr, m_builder.getInt64Ty(), "mem.index");
	auto index31 = m_builder.CreateAdd(index, llvm::ConstantInt::get(m_builder.getInt64Ty(), 31), "mem.index.31");

	// load from evmccrt_memory_require()[index]
	auto base = m_builder.CreateCall(m_memRequire, index31, "base");
	auto ptr = m_builder.CreateGEP(base, index, "ptr");

	auto i256ptrTy = m_builder.getIntNTy(256)->getPointerTo();
	auto wordPtr = m_builder.CreateBitCast(ptr, i256ptrTy, "wordptr");
	auto byte = m_builder.CreateLoad(wordPtr, "word");

	dump(0);
	return byte;
}

void Memory::storeWord(llvm::Value* _addr, llvm::Value* _word)
{
	auto index = m_builder.CreateTrunc(_addr, m_builder.getInt64Ty(), "mem.index");
	auto index31 = m_builder.CreateAdd(index, llvm::ConstantInt::get(m_builder.getInt64Ty(), 31), "mem.index31");

	auto base = m_builder.CreateCall(m_memRequire, index31, "base");
	auto ptr = m_builder.CreateGEP(base, index, "ptr");

	auto i256ptrTy = m_builder.getIntNTy(256)->getPointerTo();
	auto wordPtr = m_builder.CreateBitCast(ptr, i256ptrTy, "wordptr");
	m_builder.CreateStore(_word, wordPtr);

	dump(0);
}

void Memory::storeByte(llvm::Value* _addr, llvm::Value* _word)
{
	auto byte = m_builder.CreateTrunc(_word, m_builder.getInt8Ty(), "byte");
	auto index = m_builder.CreateTrunc(_addr, m_builder.getInt64Ty(), "index");

	auto base = m_builder.CreateCall(m_memRequire, index, "base");
	auto ptr = m_builder.CreateGEP(base, index, "ptr");
	m_builder.CreateStore(byte, ptr);

	dump(0);
}

llvm::Value* Memory::getSize()
{
	auto size = m_builder.CreateCall(m_memSize, "mem.size");
	auto word = m_builder.CreateZExt(size, m_builder.getIntNTy(256), "mem.wsize");
	return word;
}

void Memory::dump(uint64_t _begin, uint64_t _end)
{
	if (getenv("EVMCC_DEBUG_MEMORY") == nullptr)
		return;

	auto beginVal = llvm::ConstantInt::get(m_builder.getInt64Ty(), _begin);
	auto endVal = llvm::ConstantInt::get(m_builder.getInt64Ty(), _end);

	std::vector<llvm::Value*> args = {beginVal, endVal};
	m_builder.CreateCall(m_memDump, llvm::ArrayRef<llvm::Value*>(args));
}

} // namespace evmcc

extern "C"
{
	using namespace evmcc;

// Resizes memory to contain at least _index + 1 bytes and returns the base address.
EXPORT uint8_t* evmccrt_memory_require(uint64_t _index)
{
	uint64_t requiredSize = (_index / 32 + 1) * 32;

	if (evmccrt_memory->size() < requiredSize)
	{
		std::cerr << "MEMORY: current size: " << std::dec
				  << evmccrt_memory->size() << " bytes, required size: "
				  << requiredSize << " bytes"
				  << std::endl;

		evmccrt_memory->resize(requiredSize);
	}

	return evmccrt_memory->data();
}

EXPORT uint64_t evmccrt_memory_size()
{
	return evmccrt_memory->size() / 32;
}

EXPORT void evmccrt_memory_dump(uint64_t _begin, uint64_t _end)
{
	if (_end == 0)
		_end = evmccrt_memory->size();

	std::cerr << "MEMORY: active size: " << std::dec
			  << evmccrt_memory_size() << " words\n";
	std::cerr << "MEMORY: dump from " << std::dec
			  << _begin << " to " << _end << ":";
	if (_end <= _begin)
		return;

	_begin = _begin / 16 * 16;
	for (size_t i = _begin; i < _end; i++)
	{
		if ((i - _begin) % 16 == 0)
			std::cerr << '\n' << std::dec << i << ":  ";

		uint8_t b = (*evmccrt_memory)[i];
		std::cerr << std::hex << std::setw(2) << static_cast<int>(b) << ' ';
	}
	std::cerr << std::endl;
}

}	// extern "C"
