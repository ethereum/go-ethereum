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

namespace dev
{
namespace eth
{
namespace jit
{

Memory::Memory(llvm::IRBuilder<>& _builder, GasMeter& _gasMeter):
	CompilerHelper(_builder)
{
	auto module = getModule();
	auto i64Ty = m_builder.getInt64Ty();
	llvm::Type* argTypes[] = {i64Ty, i64Ty};
	auto dumpTy = llvm::FunctionType::get(m_builder.getVoidTy(), llvm::ArrayRef<llvm::Type*>(argTypes), false);
 	m_memDump = llvm::Function::Create(dumpTy, llvm::GlobalValue::LinkageTypes::ExternalLinkage,
		"evmccrt_memory_dump", module);

	m_data = new llvm::GlobalVariable(*module, Type::BytePtr, false, llvm::GlobalVariable::PrivateLinkage, llvm::UndefValue::get(Type::BytePtr), "mem.data");
	m_data->setUnnamedAddr(true); // Address is not important

	m_size = new llvm::GlobalVariable(*module, Type::i256, false, llvm::GlobalVariable::PrivateLinkage, Constant::get(0), "mem.size");
	m_size->setUnnamedAddr(true); // Address is not important

	m_returnDataOffset = new llvm::GlobalVariable(*module, Type::i256, false, llvm::GlobalVariable::ExternalLinkage, nullptr, "mem_returnDataOffset");
	m_returnDataOffset->setUnnamedAddr(true); // Address is not important

	m_returnDataSize = new llvm::GlobalVariable(*module, Type::i256, false, llvm::GlobalVariable::ExternalLinkage, nullptr, "mem_returnDataSize");
	m_returnDataSize->setUnnamedAddr(true); // Address is not important

	m_resize = llvm::Function::Create(llvm::FunctionType::get(Type::BytePtr, Type::WordPtr, false), llvm::Function::ExternalLinkage, "mem_resize", module);
	llvm::AttrBuilder attrBuilder;
	attrBuilder.addAttribute(llvm::Attribute::NoAlias).addAttribute(llvm::Attribute::NoCapture).addAttribute(llvm::Attribute::NonNull).addAttribute(llvm::Attribute::ReadOnly);
	m_resize->setAttributes(llvm::AttributeSet::get(m_resize->getContext(), 1, attrBuilder));

	m_require = createRequireFunc(_gasMeter);
	m_loadWord = createFunc(false, Type::i256, _gasMeter);
	m_storeWord = createFunc(true, Type::i256, _gasMeter);
	m_storeByte = createFunc(true, Type::Byte,  _gasMeter);
}

llvm::Function* Memory::createRequireFunc(GasMeter& _gasMeter)
{
	auto func = llvm::Function::Create(llvm::FunctionType::get(Type::Void, Type::i256, false), llvm::Function::PrivateLinkage, "mem.require", getModule());

	auto checkBB = llvm::BasicBlock::Create(func->getContext(), "check", func);
	auto resizeBB = llvm::BasicBlock::Create(func->getContext(), "resize", func);
	auto returnBB = llvm::BasicBlock::Create(func->getContext(), "return", func);

	InsertPointGuard guard(m_builder); // Restores insert point at function exit

	// BB "check"
	m_builder.SetInsertPoint(checkBB);
	llvm::Value* sizeRequired = func->arg_begin();
	sizeRequired->setName("sizeRequired");
	auto size = m_builder.CreateLoad(m_size, "size");
	auto resizeNeeded = m_builder.CreateICmpULE(size, sizeRequired, "resizeNeeded");
	m_builder.CreateCondBr(resizeNeeded, resizeBB, returnBB); // OPT branch weights?

	// BB "resize"
	m_builder.SetInsertPoint(resizeBB);
	// Check gas first
	auto wordsRequired = m_builder.CreateUDiv(m_builder.CreateAdd(sizeRequired, Constant::get(31)), Constant::get(32), "wordsRequired");
	auto words = m_builder.CreateUDiv(m_builder.CreateAdd(size, Constant::get(31)), Constant::get(32), "words");
	auto newWords = m_builder.CreateSub(wordsRequired, words, "addtionalWords");
	_gasMeter.checkMemory(newWords, m_builder);
	// Resize
	m_builder.CreateStore(sizeRequired, m_size);
	auto newData = m_builder.CreateCall(m_resize, m_size, "newData");
	m_builder.CreateStore(newData, m_data);
	m_builder.CreateBr(returnBB);

	// BB "return"
	m_builder.SetInsertPoint(returnBB);
	m_builder.CreateRetVoid();
	return func;
}

llvm::Function* Memory::createFunc(bool _isStore, llvm::Type* _valueType, GasMeter& _gasMeter)
{
	auto isWord = _valueType == Type::i256;

	llvm::Type* storeArgs[] = {Type::i256, _valueType};
	auto name = _isStore ? isWord ? "mstore" : "mstore8" : "mload";
	auto funcType = _isStore ? llvm::FunctionType::get(Type::Void, storeArgs, false) : llvm::FunctionType::get(Type::i256, Type::i256, false);
	auto func = llvm::Function::Create(funcType, llvm::Function::PrivateLinkage, name, getModule());

	InsertPointGuard guard(m_builder); // Restores insert point at function exit

	m_builder.SetInsertPoint(llvm::BasicBlock::Create(func->getContext(), {}, func));
	llvm::Value* index = func->arg_begin();
	index->setName("index");
	
	auto valueSize = _valueType->getPrimitiveSizeInBits() / 8;
	this->require(index, Constant::get(valueSize));
	auto data = m_builder.CreateLoad(m_data, "data");
	auto ptr = m_builder.CreateGEP(data, index, "ptr");
	if (isWord)
		ptr = m_builder.CreateBitCast(ptr, Type::WordPtr, "wordPtr");
	if (_isStore)
	{
		llvm::Value* value = ++func->arg_begin();
		value->setName("value");
		m_builder.CreateStore(value, ptr);
		m_builder.CreateRetVoid();
	}
	else
	{
		auto ret = m_builder.CreateLoad(ptr);
		m_builder.CreateRet(ret);
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

llvm::Value* Memory::getData()
{
	return m_builder.CreateLoad(m_data);
}

llvm::Value* Memory::getSize()
{
	return m_builder.CreateLoad(m_size);
}

void Memory::require(llvm::Value* _size)
{
	m_builder.CreateCall(m_require, _size);
}

void Memory::require(llvm::Value* _offset, llvm::Value* _size)
{
	auto sizeRequired = m_builder.CreateAdd(_offset, _size, "sizeRequired");
	require(sizeRequired);
}

void Memory::registerReturnData(llvm::Value* _index, llvm::Value* _size)
{ 
	require(_index, _size); // Make sure that memory is allocated and count gas

	m_builder.CreateStore(_index, m_returnDataOffset);
	m_builder.CreateStore(_size, m_returnDataSize);
}

void Memory::copyBytes(llvm::Value* _srcPtr, llvm::Value* _srcSize, llvm::Value* _srcIdx,
                       llvm::Value* _destMemIdx, llvm::Value* _reqBytes)
{
	auto zero256 = llvm::ConstantInt::get(Type::i256, 0);

	auto reqMemSize = m_builder.CreateAdd(_destMemIdx, _reqBytes, "req_mem_size");
	require(reqMemSize);

	auto srcPtr = m_builder.CreateGEP(_srcPtr, _srcIdx, "src_idx");

	auto memPtr = getData();
	auto destPtr = m_builder.CreateGEP(memPtr, _destMemIdx, "dest_mem_ptr");

	// remaining source bytes:
	auto remSrcSize = m_builder.CreateSub(_srcSize, _srcIdx);
	auto remSizeNegative = m_builder.CreateICmpSLT(remSrcSize, zero256);
	auto remSrcBytes = m_builder.CreateSelect(remSizeNegative, zero256, remSrcSize, "rem_src_bytes");

	auto tooFewSrcBytes = m_builder.CreateICmpULT(remSrcBytes, _reqBytes);
	auto bytesToCopy = m_builder.CreateSelect(tooFewSrcBytes, remSrcBytes, _reqBytes, "bytes_to_copy");

	m_builder.CreateMemCpy(destPtr, srcPtr, bytesToCopy, 0);
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

}
}
}


extern "C"
{

using namespace dev::eth::jit;

EXPORT i256 mem_returnDataOffset;
EXPORT i256 mem_returnDataSize;

EXPORT uint8_t* mem_resize(i256* _size)
{
	auto size = _size->a; // Trunc to 64-bit
	auto& memory = Runtime::getMemory();
	memory.resize(size);
	return memory.data();
}

EXPORT void evmccrt_memory_dump(uint64_t _begin, uint64_t _end)
{
	if (_end == 0)
		_end = Runtime::getMemory().size();

	std::cerr << "MEMORY: active size: " << std::dec
			  << Runtime::getMemory().size() / 32 << " words\n";
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

dev::bytesConstRef dev::eth::jit::Memory::getReturnData()
{
	// TODO: Handle large indexes
	auto offset = static_cast<size_t>(llvm2eth(mem_returnDataOffset));
	auto size = static_cast<size_t>(llvm2eth(mem_returnDataSize));
	auto& memory = Runtime::getMemory();
	return {memory.data() + offset, size};
}
