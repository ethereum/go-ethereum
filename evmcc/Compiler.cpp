
#include "Compiler.h"

#include <llvm/IR/IRBuilder.h>

#include <libevmface/Instruction.h>

#include "Memory.h"
#include "Stack.h"
#include "Ext.h"

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
	llvm::Type* WordLowPrecision;
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

	// TODO: Use 64-bit for now. In 128-bit compiler-rt library functions are required
	Types.WordLowPrecision = llvm::Type::getIntNTy(context, 64);
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

	// Init stack and memory
	auto stack = Stack(builder, module.get());
	auto memory = Memory(builder, module.get());

	auto ext = Ext(builder);

	for (auto pc = bytecode.cbegin(); pc != bytecode.cend(); ++pc)
	{
		using dev::eth::Instruction;

		auto inst = static_cast<Instruction>(*pc);
		switch (inst)
		{

		case Instruction::ADD:
		{
			auto lhs = stack.pop();
			auto rhs = stack.pop();
			auto result = builder.CreateAdd(lhs, rhs);
			stack.push(result);
			break;
		}

		case Instruction::SUB:
		{
			auto lhs = stack.pop();
			auto rhs = stack.pop();
			auto result = builder.CreateSub(lhs, rhs);
			stack.push(result);
			break;
		}

		case Instruction::MUL:
		{
			auto lhs256 = stack.pop();
			auto rhs256 = stack.pop();
			auto lhs128 = builder.CreateTrunc(lhs256, Types.WordLowPrecision);
			auto rhs128 = builder.CreateTrunc(rhs256, Types.WordLowPrecision);
			auto res128 = builder.CreateMul(lhs128, rhs128);
			auto res256 = builder.CreateZExt(res128, Types.word256);
			stack.push(res256);
			break;
		}

		case Instruction::DIV:
		{
			auto lhs256 = stack.pop();
			auto rhs256 = stack.pop();
			auto lhs128 = builder.CreateTrunc(lhs256, Types.WordLowPrecision);
			auto rhs128 = builder.CreateTrunc(rhs256, Types.WordLowPrecision);
			auto res128 = builder.CreateUDiv(lhs128, rhs128);
			auto res256 = builder.CreateZExt(res128, Types.word256);
			stack.push(res256);
			break;
		}

		case Instruction::SDIV:
		{
			auto lhs256 = stack.pop();
			auto rhs256 = stack.pop();
			auto lhs128 = builder.CreateTrunc(lhs256, Types.WordLowPrecision);
			auto rhs128 = builder.CreateTrunc(rhs256, Types.WordLowPrecision);
			auto res128 = builder.CreateSDiv(lhs128, rhs128);
			auto res256 = builder.CreateSExt(res128, Types.word256);
			stack.push(res256);
			break;
		}

		case Instruction::MOD:
		{
			auto lhs256 = stack.pop();
			auto rhs256 = stack.pop();
			auto lhs128 = builder.CreateTrunc(lhs256, Types.WordLowPrecision);
			auto rhs128 = builder.CreateTrunc(rhs256, Types.WordLowPrecision);
			auto res128 = builder.CreateURem(lhs128, rhs128);
			auto res256 = builder.CreateZExt(res128, Types.word256);
			stack.push(res256);
			break;
		}

		case Instruction::SMOD:
		{
			auto lhs256 = stack.pop();
			auto rhs256 = stack.pop();
			auto lhs128 = builder.CreateTrunc(lhs256, Types.WordLowPrecision);
			auto rhs128 = builder.CreateTrunc(rhs256, Types.WordLowPrecision);
			auto res128 = builder.CreateSRem(lhs128, rhs128);
			auto res256 = builder.CreateSExt(res128, Types.word256);
			stack.push(res256);
			break;
		}

		case Instruction::POP:
		{
			stack.pop();
			break;
		}

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

		case Instruction::DUP1:
		case Instruction::DUP2:
		case Instruction::DUP3:
		case Instruction::DUP4:
		case Instruction::DUP5:
		case Instruction::DUP6:
		case Instruction::DUP7:
		case Instruction::DUP8:
		case Instruction::DUP9:
		case Instruction::DUP10:
		case Instruction::DUP11:
		case Instruction::DUP12:
		case Instruction::DUP13:
		case Instruction::DUP14:
		case Instruction::DUP15:
		case Instruction::DUP16:
		{
			auto index = static_cast<uint32_t>(inst) - static_cast<uint32_t>(Instruction::DUP1);
			auto value = stack.get(index);
			stack.push(value);
			break;
		}

		case Instruction::SWAP1:
		case Instruction::SWAP2:
		case Instruction::SWAP3:
		case Instruction::SWAP4:
		case Instruction::SWAP5:
		case Instruction::SWAP6:
		case Instruction::SWAP7:
		case Instruction::SWAP8:
		case Instruction::SWAP9:
		case Instruction::SWAP10:
		case Instruction::SWAP11:
		case Instruction::SWAP12:
		case Instruction::SWAP13:
		case Instruction::SWAP14:
		case Instruction::SWAP15:
		case Instruction::SWAP16:
		{
			auto index = static_cast<uint32_t>(inst) - static_cast<uint32_t>(Instruction::SWAP1) + 1;
			auto loValue = stack.get(index);
			auto hiValue = stack.get(0);
			stack.set(index, hiValue);
			stack.set(0, loValue);
			break;
		}

		}
	}

	builder.CreateRet(ConstantInt::get(Type::getInt32Ty(context), 0));

	return module;
}

}
