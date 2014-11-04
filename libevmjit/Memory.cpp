#include "Memory.h"

#include <vector>
#include <iostream>
#include <iomanip>
#include <cstdint>
#include <cassert>

#include <llvm/IR/GlobalVariable.h>
#include <llvm/IR/Function.h>
#include <llvm/IR/IntrinsicInst.h>

#include <libdevcore/Common.h>

#include "Type.h"
#include "Runtime.h"
#include "GasMeter.h"
#include "Endianness.h"
#include "Runtime.h"

namespace dev
{
namespace eth
{
namespace jit
{

Memory::Memory(RuntimeManager& _runtimeManager, GasMeter& _gasMeter):
	RuntimeHelper(_runtimeManager)
{
	auto module = getModule();
	llvm::Type* argTypes[] = {Type::Word, Type::Word};
	auto dumpTy = llvm::FunctionType::get(m_builder.getVoidTy(), llvm::ArrayRef<llvm::Type*>(argTypes), false);
	m_memDump = llvm::Function::Create(dumpTy, llvm::GlobalValue::LinkageTypes::ExternalLinkage,
									   "evmccrt_memory_dump", module);

	m_data = new llvm::GlobalVariable(*module, Type::BytePtr, false, llvm::GlobalVariable::PrivateLinkage, llvm::UndefValue::get(Type::BytePtr), "mem.data");
	m_data->setUnnamedAddr(true); // Address is not important

	m_size = new llvm::GlobalVariable(*module, Type::Word, false, llvm::GlobalVariable::PrivateLinkage, Constant::get(0), "mem.size");
	m_size->setUnnamedAddr(true); // Address is not important

	llvm::Type* resizeArgs[] = {Type::RuntimePtr, Type::WordPtr};
	m_resize = llvm::Function::Create(llvm::FunctionType::get(Type::BytePtr, resizeArgs, false), llvm::Function::ExternalLinkage, "mem_resize", module);
	llvm::AttrBuilder attrBuilder;
	attrBuilder.addAttribute(llvm::Attribute::NoAlias).addAttribute(llvm::Attribute::NoCapture).addAttribute(llvm::Attribute::NonNull).addAttribute(llvm::Attribute::ReadOnly);
	m_resize->setAttributes(llvm::AttributeSet::get(m_resize->getContext(), 1, attrBuilder));

	m_require = createRequireFunc(_gasMeter, _runtimeManager);
	m_loadWord = createFunc(false, Type::Word, _gasMeter);
	m_storeWord = createFunc(true, Type::Word, _gasMeter);
	m_storeByte = createFunc(true, Type::Byte,  _gasMeter);
}

llvm::Function* Memory::createRequireFunc(GasMeter& _gasMeter, RuntimeManager& _runtimeManager)
{
	llvm::Type* argTypes[] = {Type::Word, Type::Word};
	auto func = llvm::Function::Create(llvm::FunctionType::get(Type::Void, argTypes, false), llvm::Function::PrivateLinkage, "mem.require", getModule());
	auto offset = func->arg_begin();
	offset->setName("offset");
	auto size = offset->getNextNode();
	size->setName("size");

	auto checkBB = llvm::BasicBlock::Create(func->getContext(), "Check", func);
	auto resizeBB = llvm::BasicBlock::Create(func->getContext(), "Resize", func);
	auto returnBB = llvm::BasicBlock::Create(func->getContext(), "Return", func);

	InsertPointGuard guard(m_builder); // Restores insert point at function exit

	// BB "Check"
	m_builder.SetInsertPoint(checkBB);
	auto uaddWO = llvm::Intrinsic::getDeclaration(getModule(), llvm::Intrinsic::uadd_with_overflow, Type::Word);
	auto uaddRes = m_builder.CreateCall2(uaddWO, offset, size, "res");
	auto sizeRequired = m_builder.CreateExtractValue(uaddRes, 0, "sizeReq");
	auto overflow1 = m_builder.CreateExtractValue(uaddRes, 1, "overflow1");
	auto currSize = m_builder.CreateLoad(m_size, "currSize");
	auto tooSmall = m_builder.CreateICmpULE(currSize, sizeRequired, "tooSmall");
	auto resizeNeeded = m_builder.CreateOr(tooSmall, overflow1, "resizeNeeded");
	m_builder.CreateCondBr(resizeNeeded, resizeBB, returnBB); // OPT branch weights?

	// BB "Resize"
	m_builder.SetInsertPoint(resizeBB);
	// Check gas first
	uaddRes = m_builder.CreateCall2(uaddWO, sizeRequired, Constant::get(31), "res");
	auto wordsRequired = m_builder.CreateExtractValue(uaddRes, 0);
	auto overflow2 = m_builder.CreateExtractValue(uaddRes, 1, "overflow2");
	auto overflow = m_builder.CreateOr(overflow1, overflow2, "overflow");
	wordsRequired = m_builder.CreateSelect(overflow, Constant::get(-1), wordsRequired);
	wordsRequired = m_builder.CreateUDiv(wordsRequired, Constant::get(32), "wordsReq");
	sizeRequired = m_builder.CreateMul(wordsRequired, Constant::get(32), "roundedSizeReq");
	auto words = m_builder.CreateUDiv(currSize, Constant::get(32), "words");	// size is always 32*k
	auto newWords = m_builder.CreateSub(wordsRequired, words, "addtionalWords");
	_gasMeter.checkMemory(newWords);
	// Resize
	m_builder.CreateStore(sizeRequired, m_size);
	auto newData = m_builder.CreateCall2(m_resize, _runtimeManager.getRuntimePtr(), m_size, "newData");
	m_builder.CreateStore(newData, m_data);
	m_builder.CreateBr(returnBB);

	// BB "Return"
	m_builder.SetInsertPoint(returnBB);
	m_builder.CreateRetVoid();
	return func;
}

llvm::Function* Memory::createFunc(bool _isStore, llvm::Type* _valueType, GasMeter&)
{
	auto isWord = _valueType == Type::Word;

	llvm::Type* storeArgs[] = {Type::Word, _valueType};
	auto name = _isStore ? isWord ? "mstore" : "mstore8" : "mload";
	auto funcType = _isStore ? llvm::FunctionType::get(Type::Void, storeArgs, false) : llvm::FunctionType::get(Type::Word, Type::Word, false);
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
		if (isWord)
			value = Endianness::toBE(m_builder, value);
		m_builder.CreateStore(value, ptr);
		m_builder.CreateRetVoid();
	}
	else
	{
		llvm::Value* ret = m_builder.CreateLoad(ptr);
		ret = Endianness::toNative(m_builder, ret);
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

void Memory::require(llvm::Value* _offset, llvm::Value* _size)
{
	m_builder.CreateCall2(m_require, _offset, _size);
}

void Memory::copyBytes(llvm::Value* _srcPtr, llvm::Value* _srcSize, llvm::Value* _srcIdx,
					   llvm::Value* _destMemIdx, llvm::Value* _reqBytes)
{
	auto zero256 = llvm::ConstantInt::get(Type::Word, 0);

	require(_destMemIdx, _reqBytes);

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

	EXPORT uint8_t* mem_resize(Runtime* _rt, i256* _size)
	{
		auto size = _size->a; // Trunc to 64-bit
		auto& memory = _rt->getMemory();
		memory.resize(size);
		return memory.data();
	}
}
