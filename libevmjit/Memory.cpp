#include "Memory.h"

#include <vector>
#include <iostream>
#include <iomanip>
#include <cstdint>
#include <cassert>

#include <llvm/IR/GlobalVariable.h>
#include <llvm/IR/Function.h>
#include <llvm/IR/IntrinsicInst.h>

#include "Type.h"
#include "Runtime.h"
#include "GasMeter.h"
#include "Endianness.h"
#include "RuntimeManager.h"

namespace dev
{
namespace eth
{
namespace jit
{

Memory::Memory(RuntimeManager& _runtimeManager, GasMeter& _gasMeter):
	RuntimeHelper(_runtimeManager),  // TODO: RuntimeHelper not needed
	m_gasMeter(_gasMeter)
{
	llvm::Type* resizeArgs[] = {Type::RuntimePtr, Type::WordPtr};
	m_resize = llvm::Function::Create(llvm::FunctionType::get(Type::BytePtr, resizeArgs, false), llvm::Function::ExternalLinkage, "mem_resize", getModule());
	llvm::AttrBuilder attrBuilder;
	attrBuilder.addAttribute(llvm::Attribute::NoAlias).addAttribute(llvm::Attribute::NoCapture).addAttribute(llvm::Attribute::NonNull).addAttribute(llvm::Attribute::ReadOnly);
	m_resize->setAttributes(llvm::AttributeSet::get(m_resize->getContext(), 1, attrBuilder));

	m_require = createRequireFunc(_gasMeter);
	m_loadWord = createFunc(false, Type::Word, _gasMeter);
	m_storeWord = createFunc(true, Type::Word, _gasMeter);
	m_storeByte = createFunc(true, Type::Byte,  _gasMeter);
}

llvm::Function* Memory::createRequireFunc(GasMeter& _gasMeter)
{
	llvm::Type* argTypes[] = {Type::RuntimePtr, Type::Word, Type::Word};
	auto func = llvm::Function::Create(llvm::FunctionType::get(Type::Void, argTypes, false), llvm::Function::PrivateLinkage, "mem.require", getModule());
	auto rt = func->arg_begin();
	rt->setName("rt");
	auto offset = rt->getNextNode();
	offset->setName("offset");
	auto size = offset->getNextNode();
	size->setName("size");

	auto preBB = llvm::BasicBlock::Create(func->getContext(), "Pre", func);
	auto checkBB = llvm::BasicBlock::Create(func->getContext(), "Check", func);
	auto resizeBB = llvm::BasicBlock::Create(func->getContext(), "Resize", func);
	auto returnBB = llvm::BasicBlock::Create(func->getContext(), "Return", func);

	InsertPointGuard guard(m_builder); // Restores insert point at function exit

	// BB "Pre": Ignore checks with size 0
	m_builder.SetInsertPoint(preBB);
	auto sizeIsZero = m_builder.CreateICmpEQ(size, Constant::get(0));
	m_builder.CreateCondBr(sizeIsZero, returnBB, checkBB);

	// BB "Check"
	m_builder.SetInsertPoint(checkBB);
	auto uaddWO = llvm::Intrinsic::getDeclaration(getModule(), llvm::Intrinsic::uadd_with_overflow, Type::Word);
	auto uaddRes = m_builder.CreateCall2(uaddWO, offset, size, "res");
	auto sizeRequired = m_builder.CreateExtractValue(uaddRes, 0, "sizeReq");
	auto overflow1 = m_builder.CreateExtractValue(uaddRes, 1, "overflow1");
	auto rtPtr = getRuntimeManager().getRuntimePtr();
	auto sizePtr = m_builder.CreateStructGEP(rtPtr, 4);
	auto currSize = m_builder.CreateLoad(sizePtr, "currSize");
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
	_gasMeter.countMemory(newWords);
	// Resize
	m_builder.CreateStore(sizeRequired, sizePtr);
	auto newData = m_builder.CreateCall2(m_resize, rt, sizePtr, "newData");
	auto dataPtr = m_builder.CreateStructGEP(rtPtr, 3);
	m_builder.CreateStore(newData, dataPtr);
	m_builder.CreateBr(returnBB);

	// BB "Return"
	m_builder.SetInsertPoint(returnBB);
	m_builder.CreateRetVoid();
	return func;
}

llvm::Function* Memory::createFunc(bool _isStore, llvm::Type* _valueType, GasMeter&)
{
	auto isWord = _valueType == Type::Word;

	llvm::Type* storeArgs[] = {Type::RuntimePtr, Type::Word, _valueType};
	llvm::Type* loadArgs[] = {Type::RuntimePtr, Type::Word};
	auto name = _isStore ? isWord ? "mstore" : "mstore8" : "mload";
	auto funcType = _isStore ? llvm::FunctionType::get(Type::Void, storeArgs, false) : llvm::FunctionType::get(Type::Word, loadArgs, false);
	auto func = llvm::Function::Create(funcType, llvm::Function::PrivateLinkage, name, getModule());

	InsertPointGuard guard(m_builder); // Restores insert point at function exit

	m_builder.SetInsertPoint(llvm::BasicBlock::Create(func->getContext(), {}, func));
	auto rt = func->arg_begin();
	rt->setName("rt");
	auto index = rt->getNextNode();
	index->setName("index");

	auto valueSize = _valueType->getPrimitiveSizeInBits() / 8;
	this->require(index, Constant::get(valueSize));
	auto ptr = getBytePtr(index);
	if (isWord)
		ptr = m_builder.CreateBitCast(ptr, Type::WordPtr, "wordPtr");
	if (_isStore)
	{
		llvm::Value* value = index->getNextNode();
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
	return createCall(m_loadWord, getRuntimeManager().getRuntimePtr(), _addr);
}

void Memory::storeWord(llvm::Value* _addr, llvm::Value* _word)
{
	createCall(m_storeWord, getRuntimeManager().getRuntimePtr(), _addr, _word);
}

void Memory::storeByte(llvm::Value* _addr, llvm::Value* _word)
{
	auto byte = m_builder.CreateTrunc(_word, Type::Byte, "byte");
	createCall(m_storeByte, getRuntimeManager().getRuntimePtr(), _addr, byte);
}

llvm::Value* Memory::getData()
{
	auto rtPtr = getRuntimeManager().getRuntimePtr();
	auto dataPtr = m_builder.CreateStructGEP(rtPtr, 3);
	return m_builder.CreateLoad(dataPtr, "data");
}

llvm::Value* Memory::getSize()
{
	auto rtPtr = getRuntimeManager().getRuntimePtr();
	auto sizePtr = m_builder.CreateStructGEP(rtPtr, 4);
	return m_builder.CreateLoad(sizePtr, "size");
}

llvm::Value* Memory::getBytePtr(llvm::Value* _index)
{
	return m_builder.CreateGEP(getData(), _index, "ptr");
}

void Memory::require(llvm::Value* _offset, llvm::Value* _size)
{
	createCall(m_require, getRuntimeManager().getRuntimePtr(), _offset, _size);
}

void Memory::copyBytes(llvm::Value* _srcPtr, llvm::Value* _srcSize, llvm::Value* _srcIdx,
					   llvm::Value* _destMemIdx, llvm::Value* _reqBytes)
{
	require(_destMemIdx, _reqBytes);

	// Additional copy cost
	// TODO: This round ups to 32 happens in many places
	auto copyWords = m_builder.CreateUDiv(m_builder.CreateAdd(_reqBytes, Constant::get(31)), Constant::get(32));
	m_gasMeter.countCopy(copyWords);

	// Algorithm:
	// isOutsideData = idx256 >= size256
	// idx64  = trunc idx256
	// size64 = trunc size256
	// dataLeftSize = size64 - idx64  // safe if not isOutsideData
	// reqBytes64 = trunc _reqBytes   // require() handles large values
	// bytesToCopy0 = select(reqBytes64 > dataLeftSize, dataSizeLeft, reqBytes64)  // min
	// bytesToCopy = select(isOutsideData, 0, bytesToCopy0)

	auto isOutsideData = m_builder.CreateICmpUGE(_srcIdx, _srcSize);
	auto idx64 = m_builder.CreateTrunc(_srcIdx, Type::lowPrecision);
	auto size64 = m_builder.CreateTrunc(_srcSize, Type::lowPrecision);
	auto dataLeftSize = m_builder.CreateNUWSub(size64, idx64);
	auto reqBytes64 = m_builder.CreateTrunc(_reqBytes, Type::lowPrecision);
	auto outOfBound = m_builder.CreateICmpUGT(reqBytes64, dataLeftSize);
	auto bytesToCopyInner = m_builder.CreateSelect(outOfBound, dataLeftSize, reqBytes64);
	auto zero64 = llvm::ConstantInt::get(Type::lowPrecision, 0);	// TODO: Cache common constants
	auto bytesToCopy = m_builder.CreateSelect(isOutsideData, zero64, bytesToCopyInner);

	auto src = m_builder.CreateGEP(_srcPtr, idx64, "src");
	auto dst = m_builder.CreateGEP(getData(), _destMemIdx, "dst");
	m_builder.CreateMemCpy(dst, src, bytesToCopy, 0);
}

}
}
}


extern "C"
{
	using namespace dev::eth::jit;

	EXPORT byte* mem_resize(Runtime* _rt, i256* _size)	// TODO: Use uint64 as size OR use realloc in LLVM IR
	{
		auto size = _size->a; // Trunc to 64-bit
		auto& memory = _rt->getMemory();
		memory.resize(size);
		return memory.data();
	}
}
