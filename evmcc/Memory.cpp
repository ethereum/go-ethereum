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


Memory::Memory(llvm::IRBuilder<>& _builder, llvm::Module* _module)
	: m_builder(_builder)
{
	auto memoryCreate = llvm::Function::Create(llvm::FunctionType::get(m_builder.getVoidTy(), false),
	                                           llvm::GlobalValue::LinkageTypes::ExternalLinkage,
	                                           "evmccrt_memory_create", _module);
	m_builder.CreateCall(memoryCreate);


	auto memRequireTy = llvm::FunctionType::get(m_builder.getInt8PtrTy(), m_builder.getInt64Ty(), false);
	m_memRequire = llvm::Function::Create(memRequireTy,
	                                      llvm::GlobalValue::LinkageTypes::ExternalLinkage,
	                                      "evmccrt_memory_require", _module);

	auto i64Ty = m_builder.getInt64Ty();
	std::vector<llvm::Type*> argTypes = {i64Ty, i64Ty};
	auto dumpTy = llvm::FunctionType::get(m_builder.getVoidTy(), llvm::ArrayRef<llvm::Type*>(argTypes), false);
 	m_memDump = llvm::Function::Create(dumpTy, llvm::GlobalValue::LinkageTypes::ExternalLinkage,
 	                                   "evmccrt_memory_dump", _module);
}


llvm::Value* Memory::loadByte(llvm::Value* _addr)
{
	// trunc _addr (an i256) to i64 index and use it to index the memory
	auto index = m_builder.CreateTrunc(_addr, m_builder.getInt64Ty(), "index");

	// load from evmccrt_memory_require()[index]
	auto base = m_builder.CreateCall(m_memRequire, index, "base");
	auto ptr = m_builder.CreateGEP(base, index, "ptr");
	auto byte = m_builder.CreateLoad(ptr, "byte");
	return byte;
}

void Memory::storeWord(llvm::Value* _addr, llvm::Value* _word)
{
	auto index = m_builder.CreateTrunc(_addr, m_builder.getInt64Ty(), "index");
	auto index32 = m_builder.CreateAdd(index, llvm::ConstantInt::get(m_builder.getInt64Ty(), 32), "index32");

	auto base = m_builder.CreateCall(m_memRequire, index32, "base");
	auto ptr = m_builder.CreateGEP(base, index, "ptr");

	auto i256ptrTy = m_builder.getIntNTy(256)->getPointerTo();
	auto wordPtr = m_builder.CreateBitCast(ptr, i256ptrTy, "wordptr");
	m_builder.CreateStore(_word, wordPtr);
}

void Memory::storeByte(llvm::Value* _addr, llvm::Value* _byte)
{
	auto index = m_builder.CreateTrunc(_addr, m_builder.getInt64Ty(), "index");

	auto base = m_builder.CreateCall(m_memRequire, index, "base");
	auto ptr = m_builder.CreateGEP(base, index, "ptr");
	m_builder.CreateStore(_byte, ptr);
}

void Memory::dump(uint64_t _begin, uint64_t _end)
{
	auto beginVal = llvm::ConstantInt::get(m_builder.getInt64Ty(), _begin);
	auto endVal = llvm::ConstantInt::get(m_builder.getInt64Ty(), _end);

	std::vector<llvm::Value*> args = {beginVal, endVal};
	m_builder.CreateCall(m_memDump, llvm::ArrayRef<llvm::Value*>(args));
}

} // namespace evmcc

extern "C"
{
	using namespace evmcc;

static MemoryImpl* evmccrt_memory;

EXPORT void evmccrt_memory_create(void)
{
	evmccrt_memory = new MemoryImpl(1);
	std::cerr << "MEMORY: create(), initial size = " << evmccrt_memory->size()
			  << std::endl;
}

// Resizes memory to contain at least _size bytes and returns the base address.
EXPORT void* evmccrt_memory_require(uint64_t _size)
{
	std::cerr << "MEMORY: require(), current size = " << evmccrt_memory->size()
			  << ", required size = " << _size
			  << std::endl;

	if (evmccrt_memory->size() < _size)
		evmccrt_memory->resize(_size);

	return evmccrt_memory->data();
}

EXPORT void evmccrt_memory_dump(uint64_t _begin, uint64_t _end)
{
	std::cerr << "Memory dump from " << std::hex << _begin << " to " << std::hex << _end << ":";
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
