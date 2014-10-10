#include "Memory.h"

#include <vector>
#include <iostream>
#include <iomanip>
#include <cstdint>
#include <cassert>

#include <llvm/IR/GlobalVariable.h>
#include <llvm/IR/Function.h>

#include <libdevcore/Common.h>

#include "Type.h"
#include "Runtime.h"

#ifdef _MSC_VER
	#define EXPORT __declspec(dllexport)
#else
	#define EXPORT
#endif

namespace evmcc
{

Memory::Memory(llvm::IRBuilder<>& _builder, llvm::Module* _module)
	: m_builder(_builder)
{
	auto voidTy	= m_builder.getVoidTy();
	auto i64Ty = m_builder.getInt64Ty();

	auto memRequireTy = llvm::FunctionType::get(m_builder.getInt8PtrTy(), i64Ty, false);
	m_memRequire = llvm::Function::Create(memRequireTy,
	                                      llvm::GlobalValue::LinkageTypes::ExternalLinkage,
										  "evmccrt_memory_require", _module);

	auto memSizeTy = llvm::FunctionType::get(i64Ty, false);
	m_memSize = llvm::Function::Create(memSizeTy,
	                                   llvm::GlobalValue::LinkageTypes::ExternalLinkage,
									   "evmccrt_memory_size", _module);

	std::vector<llvm::Type*> argTypes = {i64Ty, i64Ty};
	auto dumpTy = llvm::FunctionType::get(m_builder.getVoidTy(), llvm::ArrayRef<llvm::Type*>(argTypes), false);
 	m_memDump = llvm::Function::Create(dumpTy, llvm::GlobalValue::LinkageTypes::ExternalLinkage,
		"evmccrt_memory_dump", _module);

	m_data = new llvm::GlobalVariable(*_module, Type::BytePtr, false, llvm::GlobalVariable::PrivateLinkage, llvm::UndefValue::get(Type::BytePtr), "mem.data");
	m_data->setUnnamedAddr(true); // Address is not important

	m_size = new llvm::GlobalVariable(*_module, Type::i256, false, llvm::GlobalVariable::PrivateLinkage, m_builder.getIntN(256, 0), "mem.size");
	m_size->setUnnamedAddr(true); // Address is not important

	m_resize = llvm::Function::Create(llvm::FunctionType::get(Type::BytePtr, Type::WordPtr, false), llvm::Function::ExternalLinkage, "mem_resize", _module);

	llvm::Type* storeArgs[] = {Type::i256, Type::i256};
	m_store = llvm::Function::Create(llvm::FunctionType::get(Type::Void, storeArgs, false), llvm::Function::PrivateLinkage, "mem.store", _module);
	auto origBB = m_builder.GetInsertBlock();
	auto origPt = m_builder.GetInsertPoint();

	auto checkBB = llvm::BasicBlock::Create(m_store->getContext(), "check", m_store);
	auto resizeBB = llvm::BasicBlock::Create(m_store->getContext(), "resize", m_store);
	auto storeBB = llvm::BasicBlock::Create(m_store->getContext(), "store", m_store);

	m_builder.SetInsertPoint(checkBB);
	llvm::Value* index = m_store->arg_begin();
	index->setName("index");
	llvm::Value* value = ++m_store->arg_begin();
	value->setName("value");
	auto sizeRequired = m_builder.CreateAdd(index, m_builder.getIntN(256, 32), "sizeRequired");
	auto size = m_builder.CreateLoad(m_size, "size");
	auto resizeNeeded = m_builder.CreateICmpULE(sizeRequired, size, "resizeNeeded");
	m_builder.CreateCondBr(resizeNeeded, resizeBB, storeBB); // OPT branch weights?

	m_builder.SetInsertPoint(resizeBB);
	m_builder.CreateStore(sizeRequired, m_size);
	llvm::Value* data = m_builder.CreateCall(m_resize, m_size, "data");
	m_builder.CreateStore(data, m_data);
	m_builder.CreateBr(storeBB);

	m_builder.SetInsertPoint(storeBB);
	data = m_builder.CreateLoad(m_data, "data");
	auto ptr = m_builder.CreateGEP(data, index, "ptr");
	ptr = m_builder.CreateBitCast(ptr, Type::WordPtr, "wordPtr");
	m_builder.CreateStore(value, ptr);
	m_builder.CreateRetVoid();

	m_builder.SetInsertPoint(origBB, origPt);
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
	m_builder.CreateCall2(m_store, _addr, _word);

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

EXPORT uint8_t* mem_resize(i256* _size)
{
	assert(false);
	auto size = _size->a; // Trunc to 64-bit
	auto& memory = Runtime::getMemory();
	memory.resize(size);
	return memory.data();
}

// Resizes memory to contain at least _index + 1 bytes and returns the base address.
EXPORT uint8_t* evmccrt_memory_require(uint64_t _index)
{
	uint64_t requiredSize = (_index / 32 + 1) * 32;
	auto&& memory = Runtime::getMemory();

	if (memory.size() < requiredSize)
	{
		std::cerr << "MEMORY: current size: " << std::dec
				  << memory.size() << " bytes, required size: "
				  << requiredSize << " bytes"
				  << std::endl;

		memory.resize(requiredSize);
	}

	return memory.data();
}

EXPORT uint64_t evmccrt_memory_size()
{
	return Runtime::getMemory().size() / 32;
}

EXPORT void evmccrt_memory_dump(uint64_t _begin, uint64_t _end)
{
	if (_end == 0)
		_end = Runtime::getMemory().size();

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

		auto b = Runtime::getMemory()[i];
		std::cerr << std::hex << std::setw(2) << static_cast<int>(b) << ' ';
	}
	std::cerr << std::endl;
}

}	// extern "C"
