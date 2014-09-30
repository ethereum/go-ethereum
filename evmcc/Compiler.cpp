
#include "Compiler.h"

#include <llvm/IR/IRBuilder.h>

#include "Memory.h"
#include "Stack.h"

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

	// Create main function
	FunctionType* funcType = FunctionType::get(llvm::Type::getInt32Ty(context), false);
	Function* mainFunc = Function::Create(funcType, Function::ExternalLinkage, "main", module.get());

	BasicBlock* entryBlock = BasicBlock::Create(context, "entry", mainFunc);
	builder.SetInsertPoint(entryBlock);


	auto stack = Stack(builder, module.get());
	auto memory = Memory(builder, module.get());

	uint64_t words[] = { 1, 2, 3, 4 };
	auto val = llvm::APInt(256, 4, words);
	auto c = ConstantInt::get(Types.word256, val);

	stack.push(c);
	stack.push(ConstantInt::get(Types.word256, 0x1122334455667788));

	auto top = stack.top();
	stack.push(top);	// dup
	stack.pop();

	auto index = ConstantInt::get(Types.word256, 123);
	memory.storeWord(index, c);

	memory.dump(123, 123+32);

	auto index2 = ConstantInt::get(Types.word256, 123 + 16);
	auto byte = memory.loadByte(index2);
	auto result = builder.CreateZExt(byte, builder.getInt32Ty());
	builder.CreateRet(result); // should return 3

	return module;
}

}
