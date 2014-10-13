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
#include "GasMeter.h"

#ifdef _MSC_VER
	#define EXPORT __declspec(dllexport)
#else
	#define EXPORT
#endif

namespace evmcc
{

Memory::Memory(llvm::IRBuilder<>& _builder, llvm::Module* _module, GasMeter& _gasMeter):
	m_builder(_builder)
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
	m_loadWord = createFunc(false, Type::i256, _module, _gasMeter);
	m_storeWord = createFunc(true, Type::i256, _module, _gasMeter);
	m_storeByte = createFunc(true, Type::Byte, _module, _gasMeter);
}

llvm::Function* Memory::createFunc(bool _isStore, llvm::Type* _valueType, llvm::Module* _module, GasMeter& _gasMeter)
{
	auto isWord = _valueType == Type::i256;

	llvm::Type* storeArgs[] = {Type::i256, _valueType};
	auto name = _isStore ? isWord ? "mstore" : "mstore8" : "mload";
	auto funcType = _isStore ? llvm::FunctionType::get(Type::Void, storeArgs, false) : llvm::FunctionType::get(Type::i256, Type::i256, false);
	auto func = llvm::Function::Create(funcType, llvm::Function::PrivateLinkage, name, _module);

	auto checkBB = llvm::BasicBlock::Create(func->getContext(), "check", func);
	auto resizeBB = llvm::BasicBlock::Create(func->getContext(), "resize", func);
	auto accessBB = llvm::BasicBlock::Create(func->getContext(), "access", func);

	// BB "check"
	llvm::IRBuilder<> builder(checkBB);
	llvm::Value* index = func->arg_begin();
	index->setName("index");
	auto valueSize = _valueType->getPrimitiveSizeInBits() / 8;
	auto sizeRequired = builder.CreateAdd(index, builder.getIntN(256, valueSize), "sizeRequired");
	auto size = builder.CreateLoad(m_size, "size");
	auto resizeNeeded = builder.CreateICmpULE(size, sizeRequired, "resizeNeeded");
	builder.CreateCondBr(resizeNeeded, resizeBB, accessBB); // OPT branch weights?

	// BB "resize"
	builder.SetInsertPoint(resizeBB);
	// Check gas first
	auto wordsRequired = builder.CreateUDiv(builder.CreateAdd(sizeRequired, builder.getIntN(256, 31)), builder.getIntN(256, 32), "wordsRequired");
	auto words = builder.CreateUDiv(builder.CreateAdd(size, builder.getIntN(256, 31)), builder.getIntN(256, 32), "words");
	auto newWords = builder.CreateSub(wordsRequired, words, "addtionalWords");
	_gasMeter.checkMemory(newWords, builder);
	// Resize
	builder.CreateStore(sizeRequired, m_size);
	auto newData = builder.CreateCall(m_resize, m_size, "newData");
	builder.CreateStore(newData, m_data);
	builder.CreateBr(accessBB);

	// BB "access"
	builder.SetInsertPoint(accessBB);
	auto data = builder.CreateLoad(m_data, "data");
	auto ptr = builder.CreateGEP(data, index, "ptr");
	if (isWord)
		ptr = builder.CreateBitCast(ptr, Type::WordPtr, "wordPtr");
	if (_isStore)
	{
		llvm::Value* value = ++func->arg_begin();
		value->setName("value");
		builder.CreateStore(value, ptr);
		builder.CreateRetVoid();
	}
	else
	{
		auto ret = builder.CreateLoad(ptr);
		builder.CreateRet(ret);
	}
	return func;
}


llvm::Value* Memory::loadWord(llvm::Value* _addr)
{
	auto value = m_builder.CreateCall(m_loadWord, _addr);

	dump(0);
	return value;
}

void Memory::storeWord(llvm::Value* _addr, llvm::Value* _word)
{
	m_builder.CreateCall2(m_storeWord, _addr, _word);

	dump(0);
}

void Memory::storeByte(llvm::Value* _addr, llvm::Value* _word)
{
	auto byte = m_builder.CreateTrunc(_word, Type::Byte, "byte");
	m_builder.CreateCall2(m_storeByte, _addr, byte);

	dump(0);
}

llvm::Value* Memory::getSize()
{
	return m_builder.CreateLoad(m_size);
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
