
#include "Compiler.h"

#include <llvm/IR/IRBuilder.h>

#include <libevmface/Instruction.h>

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
	auto mainFuncType = FunctionType::get(llvm::Type::getInt32Ty(context), false);
	auto mainFunc = Function::Create(mainFuncType, Function::ExternalLinkage, "main", module.get());

	auto entryBlock = BasicBlock::Create(context, "entry", mainFunc);
	builder.SetInsertPoint(entryBlock);

	// Init stack
	auto stack = Stack(builder, module.get());

	for (auto pc = bytecode.cbegin(); pc != bytecode.cend(); ++pc)
	{
		using dev::eth::Instruction;

		auto inst = static_cast<Instruction>(*pc);
		switch (inst)
		{
		case Instruction::PUSH1:
		case Instruction::PUSH2:
		case Instruction::PUSH3:
		case Instruction::PUSH4:
		case Instruction::PUSH5:
		case Instruction::PUSH6:
		case Instruction::PUSH7:
		case Instruction::PUSH8:
		case Instruction::PUSH9:
		case Instruction::PUSH10:
		case Instruction::PUSH11:
		case Instruction::PUSH12:
		case Instruction::PUSH13:
		case Instruction::PUSH14:
		case Instruction::PUSH15:
		case Instruction::PUSH16:
		case Instruction::PUSH17:
		case Instruction::PUSH18:
		case Instruction::PUSH19:
		case Instruction::PUSH20:
		case Instruction::PUSH21:
		case Instruction::PUSH22:
		case Instruction::PUSH23:
		case Instruction::PUSH24:
		case Instruction::PUSH25:
		case Instruction::PUSH26:
		case Instruction::PUSH27:
		case Instruction::PUSH28:
		case Instruction::PUSH29:
		case Instruction::PUSH30:
		case Instruction::PUSH31:
		case Instruction::PUSH32:
		{
			auto numBytes = static_cast<size_t>(inst) - static_cast<size_t>(Instruction::PUSH1) + 1;
			auto value = llvm::APInt(256, 0);
			for (decltype(numBytes) i = 0; i < numBytes; ++i)	// TODO: Use pc as iterator
			{
				++pc;
				value <<= 8;
				value |= *pc;
			}
			auto c = builder.getInt(value);
			stack.push(c);
			break;
		}
		}
	}

	//uint64_t words[] = { 1, 2, 3, 4 };
	//auto val = llvm::APInt(256, 4, words);
	//auto c = ConstantInt::get(Types.word256, val);

	//stack.push(c);
	//stack.push(ConstantInt::get(Types.word256, 0x1122334455667788));

	//auto top = stack.top();
	//stack.push(top);	// dup

	//stack.pop();

	builder.CreateRet(ConstantInt::get(Type::getInt32Ty(context), 0));

	return module;
}

}