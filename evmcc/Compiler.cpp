
#include "Compiler.h"

#include <llvm/IR/IRBuilder.h>

namespace evmcc
{

struct
{
	llvm::Type* word8;
	llvm::Type* word8ptr;
	llvm::Type* word256;
	llvm::Type* word256ptr;
	llvm::Type* word256arr;
	llvm::Type* size;
	llvm::Type* Void;
} Types;

Compiler::Compiler()
{
	auto& context = llvm::getGlobalContext();
	Types.word8 = llvm::Type::getInt8Ty(context);
	Types.word8ptr = llvm::Type::getInt8PtrTy(context);
	Types.word256 = llvm::Type::getIntNTy(context, 256);
	Types.word256ptr = Types.word256->getPointerTo();
	Types.word256arr = llvm::ArrayType::get(Types.word256, 100);
	Types.size = llvm::Type::getInt64Ty(context);
	Types.Void = llvm::Type::getVoidTy(context);
}


std::unique_ptr<llvm::Module> Compiler::compile(const dev::bytes& bytecode)
{
	using namespace llvm;

	auto& context = getGlobalContext();

	auto module = std::make_unique<Module>("main", context);
	IRBuilder<> builder(context);

	// Create globals for memory, memory size, stack and stack top
	auto memory = new GlobalVariable(*module, Types.word8ptr, false,
		GlobalValue::LinkageTypes::PrivateLinkage,
		Constant::getNullValue(Types.word8ptr), "memory");
	auto memSize = new GlobalVariable(*module, Types.size, false,
		GlobalValue::LinkageTypes::PrivateLinkage,
		ConstantInt::get(Types.size, 0), "memsize");
	auto stack2 = new GlobalVariable(*module, Types.word256arr, false,
		GlobalValue::LinkageTypes::PrivateLinkage,
		ConstantAggregateZero::get(Types.word256arr), "stack");
	auto stackTop = new GlobalVariable(*module, Types.size, false,
		GlobalValue::LinkageTypes::PrivateLinkage,
		ConstantInt::get(Types.size, 0), "stackTop");

	// Create value for void* malloc(size_t)
	auto mallocVal = Function::Create(FunctionType::get(Types.word8ptr, { Types.size }, false),
		GlobalValue::LinkageTypes::ExternalLinkage, "malloc", module.get());

	// Create stack_create declaration
	auto stackCreate = Function::Create(FunctionType::get(Types.word8ptr, false),
		GlobalValue::LinkageTypes::ExternalLinkage, "evmccrt_stack_create", module.get());

	auto stackPush = Function::Create(FunctionType::get(Types.Void, std::vector<Type*>{ Types.word8ptr, Types.word256 }, false),
		GlobalValue::LinkageTypes::ExternalLinkage, "evmccrt_stack_push", module.get());

	// Create main function
	FunctionType* funcType = FunctionType::get(llvm::Type::getInt32Ty(context), false);
	Function* mainFunc = Function::Create(funcType, Function::ExternalLinkage, "main", module.get());

	BasicBlock* entryBlock = BasicBlock::Create(context, "entry", mainFunc);
	builder.SetInsertPoint(entryBlock);

	// Initialize memory with call to malloc, update memsize
	std::vector<Value*> mallocMemArgs = { ConstantInt::get(Types.size, 100) };
	auto mallocMemCall = builder.CreateCall(mallocVal, mallocMemArgs, "malloc_mem");
	builder.CreateStore(mallocMemCall, memory);
	builder.CreateStore(ConstantInt::get(Types.size, 100), memSize);

	auto stack = builder.CreateCall(stackCreate, "stack");

	uint64_t words[] = { 1, 2, 3, 4 };
	auto val = llvm::APInt(256, 4, words);
	auto c = ConstantInt::get(Types.word256, val);

	Value* args[] = { stack, c };
	builder.CreateCall(stackPush, args);

	/*
	std::vector<Value*> mallocStackArgs = { ConstantInt::get(sizeTy, 200) };
	auto mallocStackCall = builder.CreateCall(mallocVal, mallocStackArgs, "malloc_stack");
	auto mallocCast = builder.CreatePointerBitCastOrAddrSpaceCast(mallocStackCall, int256ptr);
	builder.CreateStore(mallocCast, stackVal);
	*/

	builder.CreateRet(ConstantInt::get(Type::getInt32Ty(context), 13));

	return module;
}

}