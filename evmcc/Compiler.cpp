
#include "Compiler.h"

#include <boost/dynamic_bitset.hpp>

#include <llvm/IR/IRBuilder.h>
#include <llvm/IR/CFG.h>

#include <libevmface/Instruction.h>

#include "Type.h"
#include "Memory.h"
#include "Ext.h"
#include "GasMeter.h"

namespace evmcc
{

using dev::eth::Instruction;
using namespace dev::eth; // We should move all the JIT code into dev::eth namespace


Compiler::Compiler()
	: m_finalBlock(nullptr)
	, m_badJumpBlock(nullptr)
{
	Type::init(llvm::getGlobalContext());
}

void Compiler::createBasicBlocks(const dev::bytes& bytecode)
{
	std::set<ProgramCounter> splitPoints; // Sorted collections of instruction indices where basic blocks start/end
	splitPoints.insert(0);	// First basic block

	std::map<ProgramCounter, ProgramCounter> directJumpTargets;
	std::vector<ProgramCounter> indirectJumpTargets;
	boost::dynamic_bitset<> validJumpTargets(bytecode.size());

	for (auto curr = bytecode.cbegin(); curr != bytecode.cend(); ++curr)
	{
		using dev::eth::Instruction;

		ProgramCounter currentPC = curr - bytecode.cbegin();
		validJumpTargets[currentPC] = 1;

		auto inst = static_cast<Instruction>(*curr);
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
			auto next = curr + numBytes + 1;
			if (next >= bytecode.cend())
				break;

			auto nextInst = static_cast<Instruction>(*next);

			if (nextInst == Instruction::JUMP || nextInst == Instruction::JUMPI)
			{
				// Compute target PC of the jump.
				dev::u256 val = 0;
				for (auto iter = curr + 1; iter < next; ++iter)
				{
					val <<= 8;
					val |= *iter;
				}

				// Create a block for the JUMP target.
				ProgramCounter targetPC = val.convert_to<ProgramCounter>();
				if (targetPC > bytecode.size())
					targetPC = bytecode.size();
				splitPoints.insert(targetPC);

				ProgramCounter jumpPC = (next - bytecode.cbegin());
				directJumpTargets[jumpPC] = targetPC;
			}

			curr += numBytes;
			break;
		}

		case Instruction::JUMPDEST:
		{
			// A basic block starts here.
			splitPoints.insert(currentPC);
			indirectJumpTargets.push_back(currentPC);
			break;
		}

		case Instruction::JUMP:
		case Instruction::JUMPI:
		case Instruction::RETURN:
		case Instruction::STOP:
		case Instruction::SUICIDE:
		{
			// Create a basic block starting at the following instruction.
			if (curr + 1 < bytecode.cend())
			{
				splitPoints.insert(currentPC + 1);
			}
			break;
		}

		default:
			break;
		}
	}

	for (auto it = splitPoints.cbegin(); it != splitPoints.cend() && *it < bytecode.size();)
	{
		auto beginInstIdx = *it;
		++it;
		auto endInstIdx = it != splitPoints.cend() ? *it : bytecode.size();
		basicBlocks.emplace(std::piecewise_construct, std::forward_as_tuple(beginInstIdx), std::forward_as_tuple(beginInstIdx, endInstIdx, m_mainFunc));
	}

	m_finalBlock = std::make_unique<BasicBlock>("FinalBlock", m_mainFunc);
	m_badJumpBlock = std::make_unique<BasicBlock>("BadJumpBlock", m_mainFunc);

	for (auto it = directJumpTargets.cbegin(); it != directJumpTargets.cend(); ++it)
	{
		if (it->second >= bytecode.size()) // Jump out of code
		{
			m_directJumpTargets[it->first] = m_finalBlock.get();
		}
		else if (!validJumpTargets[it->second]) // Jump into data
		{
			std::cerr << "Bad JUMP at PC " << it->first
					  << ": " << it->second << " is not a valid PC\n";
			m_directJumpTargets[it->first] = m_badJumpBlock.get();
		}
		else
		{
			m_directJumpTargets[it->first] = &basicBlocks.find(it->second)->second;
		}
	}

	for (auto it = indirectJumpTargets.cbegin(); it != indirectJumpTargets.cend(); ++it)
	{
		if (*it >= bytecode.size())
			m_indirectJumpTargets.push_back(m_finalBlock.get());
		else
			m_indirectJumpTargets.push_back(&basicBlocks.find(*it)->second);
	}
}

std::unique_ptr<llvm::Module> Compiler::compile(const dev::bytes& bytecode)
{
	using namespace llvm;

	auto& context = getGlobalContext();
	auto module = std::make_unique<Module>("main", context);
	IRBuilder<> builder(context);

	// Create main function
	const auto i32Ty = builder.getInt32Ty();
	//Type* retTypeElems[] = {i32Ty, i32Ty};
	//auto retType = StructType::create(retTypeElems, "MemRef", true);
	m_mainFunc = Function::Create(FunctionType::get(builder.getInt64Ty(), false), Function::ExternalLinkage, "main", module.get());

	// Create the basic blocks.
	auto entryBlock = llvm::BasicBlock::Create(context, "entry", m_mainFunc);
	builder.SetInsertPoint(entryBlock);
	createBasicBlocks(bytecode);

	// Init runtime structures.
	auto memory = Memory(builder, module.get());
	auto ext = Ext(builder, module.get());
	GasMeter gasMeter(builder, module.get());

	// Jump to first instruction
	builder.CreateBr(basicBlocks.begin()->second);

	for (auto basicBlockPairIt = basicBlocks.begin(); basicBlockPairIt != basicBlocks.end(); ++basicBlockPairIt)
	{
		auto& basicBlock = basicBlockPairIt->second;
		auto& stack = basicBlock.getStack();
		builder.SetInsertPoint(basicBlock);

		for (auto currentPC = basicBlock.begin(); currentPC != basicBlock.end(); ++currentPC)
		{
			auto inst = static_cast<Instruction>(bytecode[currentPC]);

			// Disable for now
			//gasMeter.check(inst);

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
				auto lhs128 = builder.CreateTrunc(lhs256, Type::lowPrecision);
				auto rhs128 = builder.CreateTrunc(rhs256, Type::lowPrecision);
				auto res128 = builder.CreateMul(lhs128, rhs128);
				auto res256 = builder.CreateZExt(res128, Type::i256);
				stack.push(res256);
				break;
			}

			case Instruction::DIV:
			{
				auto lhs256 = stack.pop();
				auto rhs256 = stack.pop();
				auto lhs128 = builder.CreateTrunc(lhs256, Type::lowPrecision);
				auto rhs128 = builder.CreateTrunc(rhs256, Type::lowPrecision);
				auto res128 = builder.CreateUDiv(lhs128, rhs128);
				auto res256 = builder.CreateZExt(res128, Type::i256);
				stack.push(res256);
				break;
			}

			case Instruction::SDIV:
			{
				auto lhs256 = stack.pop();
				auto rhs256 = stack.pop();
				auto lhs128 = builder.CreateTrunc(lhs256, Type::lowPrecision);
				auto rhs128 = builder.CreateTrunc(rhs256, Type::lowPrecision);
				auto res128 = builder.CreateSDiv(lhs128, rhs128);
				auto res256 = builder.CreateSExt(res128, Type::i256);
				stack.push(res256);
				break;
			}

			case Instruction::MOD:
			{
				auto lhs256 = stack.pop();
				auto rhs256 = stack.pop();
				auto lhs128 = builder.CreateTrunc(lhs256, Type::lowPrecision);
				auto rhs128 = builder.CreateTrunc(rhs256, Type::lowPrecision);
				auto res128 = builder.CreateURem(lhs128, rhs128);
				auto res256 = builder.CreateZExt(res128, Type::i256);
				stack.push(res256);
				break;
			}

			case Instruction::SMOD:
			{
				auto lhs256 = stack.pop();
				auto rhs256 = stack.pop();
				auto lhs128 = builder.CreateTrunc(lhs256, Type::lowPrecision);
				auto rhs128 = builder.CreateTrunc(rhs256, Type::lowPrecision);
				auto res128 = builder.CreateSRem(lhs128, rhs128);
				auto res256 = builder.CreateSExt(res128, Type::i256);
				stack.push(res256);
				break;
			}

			case Instruction::EXP:
			{
				auto left = stack.pop();
				auto right = stack.pop();
				auto ret = ext.exp(left, right);
				stack.push(ret);
				break;
			}

			case Instruction::NEG:
			{
				auto top = stack.pop();
				auto zero = ConstantInt::get(Type::i256, 0);
				auto res = builder.CreateSub(zero, top);
				stack.push(res);
				break;
			}

			case Instruction::LT:
			{
				auto lhs = stack.pop();
				auto rhs = stack.pop();
				auto res1 = builder.CreateICmpULT(lhs, rhs);
				auto res256 = builder.CreateZExt(res1, Type::i256);
				stack.push(res256);
				break;
			}

			case Instruction::GT:
			{
				auto lhs = stack.pop();
				auto rhs = stack.pop();
				auto res1 = builder.CreateICmpUGT(lhs, rhs);
				auto res256 = builder.CreateZExt(res1, Type::i256);
				stack.push(res256);
				break;
			}

			case Instruction::SLT:
			{
				auto lhs = stack.pop();
				auto rhs = stack.pop();
				auto res1 = builder.CreateICmpSLT(lhs, rhs);
				auto res256 = builder.CreateZExt(res1, Type::i256);
				stack.push(res256);
				break;
			}

			case Instruction::SGT:
			{
				auto lhs = stack.pop();
				auto rhs = stack.pop();
				auto res1 = builder.CreateICmpSGT(lhs, rhs);
				auto res256 = builder.CreateZExt(res1, Type::i256);
				stack.push(res256);
				break;
			}

			case Instruction::EQ:
			{
				auto lhs = stack.pop();
				auto rhs = stack.pop();
				auto res1 = builder.CreateICmpEQ(lhs, rhs);
				auto res256 = builder.CreateZExt(res1, Type::i256);
				stack.push(res256);
				break;
			}

			case Instruction::NOT:
			{
				auto top = stack.pop();
				auto zero = ConstantInt::get(Type::i256, 0);
				auto iszero = builder.CreateICmpEQ(top, zero, "iszero");
				auto result = builder.CreateZExt(iszero, Type::i256);
				stack.push(result);
				break;
			}

			case Instruction::AND:
			{
				auto lhs = stack.pop();
				auto rhs = stack.pop();
				auto res = builder.CreateAnd(lhs, rhs);
				stack.push(res);
				break;
			}

			case Instruction::OR:
			{
				auto lhs = stack.pop();
				auto rhs = stack.pop();
				auto res = builder.CreateOr(lhs, rhs);
				stack.push(res);
				break;
			}

			case Instruction::XOR:
			{
				auto lhs = stack.pop();
				auto rhs = stack.pop();
				auto res = builder.CreateXor(lhs, rhs);
				stack.push(res);
				break;
			}

			case Instruction::BYTE:
			{
				const auto byteNum = stack.pop();
				auto value = stack.pop();

				/*
				if (byteNum < 32)	- use select
				{
				value <<= byteNum*8
				value >>= 31*8
				push value
				}
				else push 0
				*/

				// TODO: Shifting by 0 gives wrong results as of this bug http://llvm.org/bugs/show_bug.cgi?id=16439

				auto shbits = builder.CreateShl(byteNum, builder.getIntN(256, 3));
				value = builder.CreateShl(value, shbits);
				value = builder.CreateLShr(value, builder.getIntN(256, 31 * 8));

				auto byteNumValid = builder.CreateICmpULT(byteNum, builder.getIntN(256, 32));
				value = builder.CreateSelect(byteNumValid, value, builder.getIntN(256, 0));
				stack.push(value);

				break;
			}

			case Instruction::SHA3:
			{
				auto inOff = stack.pop();
				auto inSize = stack.pop();
				auto hash = ext.sha3(inOff, inSize);
				stack.push(hash);
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
				auto numBytes = static_cast<size_t>(inst)-static_cast<size_t>(Instruction::PUSH1) + 1;
				auto value = llvm::APInt(256, 0);
				for (decltype(numBytes) i = 0; i < numBytes; ++i)	// TODO: Use pc as iterator
				{
					++currentPC;
					value <<= 8;
					value |= bytecode[currentPC];
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
				auto index = static_cast<size_t>(inst)-static_cast<size_t>(Instruction::DUP1);
				stack.dup(index);
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
				auto index = static_cast<size_t>(inst)-static_cast<size_t>(Instruction::SWAP1) + 1;
				stack.swap(index);
				break;
			}

			case Instruction::MLOAD:
			{
				auto addr = stack.pop();
				auto word = memory.loadWord(addr);
				stack.push(word);
				break;
			}

			case Instruction::MSTORE:
			{
				auto addr = stack.pop();
				auto word = stack.pop();
				memory.storeWord(addr, word);
				break;
			}

			case Instruction::MSTORE8:
			{
				auto addr = stack.pop();
				auto word = stack.pop();
				memory.storeByte(addr, word);
				break;
			}

			case Instruction::MSIZE:
			{
				auto word = memory.getSize();
				stack.push(word);
				break;
			}

			case Instruction::SLOAD:
			{
				auto index = stack.pop();
				auto value = ext.store(index);
				stack.push(value);
				break;
			}

			case Instruction::SSTORE:
			{
				auto index = stack.pop();
				auto value = stack.pop();
				ext.setStore(index, value);
				break;
			}

			case Instruction::JUMP:
			case Instruction::JUMPI:
			{
				// Generate direct jump iff:
				// 1. this is not the first instruction in the block
				// 2. m_directJumpTargets[currentPC] is defined (meaning that the previous instruction is a PUSH)
				// Otherwise generate a indirect jump (a switch).
				if (currentPC != basicBlock.begin())
				{
					auto pairIter = m_directJumpTargets.find(currentPC);
					if (pairIter != m_directJumpTargets.end())
					{
						auto targetBlock = pairIter->second;

						// The target address is computed at compile time,
						// just pop it without looking...
						stack.pop();

						if (inst == Instruction::JUMP)
						{
							builder.CreateBr(targetBlock->llvm());
						}
						else // JUMPI
						{
							auto top = stack.pop();
							auto zero = ConstantInt::get(Type::i256, 0);
							auto cond = builder.CreateICmpNE(top, zero, "nonzero");

							// Assume the basic blocks are properly ordered:
							auto nextBBIter = basicBlockPairIt;
							++nextBBIter;
							assert (nextBBIter != basicBlocks.end());
							auto& followBlock = nextBBIter->second;
							builder.CreateCondBr(cond, targetBlock->llvm(), followBlock.llvm());
						}
						break;
					}
				}

				if (inst == Instruction::JUMPI)
				{
					std::cerr << "Indirect JUMPI is not supported yet (at PC "
							  << currentPC << ")\n";
					std::exit(1);
				}

				// Generate switch for indirect jump.
				auto dest = stack.pop();
				auto switchInstr = 	builder.CreateSwitch(dest, m_badJumpBlock->llvm(),
				                   	                     m_indirectJumpTargets.size());
				for (auto it = m_indirectJumpTargets.cbegin(); it != m_indirectJumpTargets.cend(); ++it)
				{
					auto& bb = *it;
					auto dest = ConstantInt::get(Type::i256, bb->begin());
					switchInstr->addCase(dest, bb->llvm());
				}

				break;
			}

			case Instruction::JUMPDEST:
			{
				// Extra asserts just in case.
				assert(currentPC == basicBlock.begin());
				break;
			}

			case Instruction::PC:
			{
				auto value = builder.getIntN(256, currentPC);
				stack.push(value);
				break;
			}

			case Instruction::ADDRESS:
			{
				auto value = ext.address();
				stack.push(value);
				break;
			}

			case Instruction::BALANCE:
			{
				auto address = stack.pop();
				auto value = ext.balance(address);
				stack.push(value);
				break;
			}

			case Instruction::CALLER:
			{
				auto value = ext.caller();
				stack.push(value);
				break;
			}

			case Instruction::ORIGIN:
			{
				auto value = ext.origin();
				stack.push(value);
				break;
			}

			case Instruction::CALLVALUE:
			{
				auto value = ext.callvalue();
				stack.push(value);
				break;
			}

			case Instruction::CALLDATASIZE:
			{
				auto value = ext.calldatasize();
				stack.push(value);
				break;
			}

			case Instruction::CALLDATALOAD:
			{
				auto index = stack.pop();
				auto value = ext.calldataload(index);
				stack.push(value);
				break;
			}

			case Instruction::GASPRICE:
			{
				auto value = ext.gasprice();
				stack.push(value);
				break;
			}

			case Instruction::CODESIZE:
			{
				auto value = builder.getIntN(256, bytecode.size());
				stack.push(value);
				break;
			}

			case Instruction::PREVHASH:
			{
				auto value = ext.prevhash();
				stack.push(value);
				break;
			}

			case Instruction::COINBASE:
			{
				auto value = ext.coinbase();
				stack.push(value);
				break;
			}

			case Instruction::TIMESTAMP:
			{
				auto value = ext.timestamp();
				stack.push(value);
				break;
			}

			case Instruction::NUMBER:
			{
				auto value = ext.number();
				stack.push(value);
				break;
			}

			case Instruction::DIFFICULTY:
			{
				auto value = ext.difficulty();
				stack.push(value);
				break;
			}

			case Instruction::GASLIMIT:
			{
				auto value = ext.gaslimit();
				stack.push(value);
				break;
			}

			case Instruction::CREATE:
			{
				auto endowment = stack.pop();
				auto initOff = stack.pop();
				auto initSize = stack.pop();

				auto address = ext.create(endowment, initOff, initSize);
				stack.push(address);
				break;
			}

			case Instruction::CALL:
			{
				auto gas = stack.pop();
				auto receiveAddress = stack.pop();
				auto value = stack.pop();
				auto inOff = stack.pop();
				auto inSize = stack.pop();
				auto outOff = stack.pop();
				auto outSize = stack.pop();

				auto ret = ext.call(gas, receiveAddress, value, inOff, inSize, outOff, outSize);
				stack.push(ret);
				break;
			}

			case Instruction::RETURN:
			{
				auto index = stack.pop();
				auto size = stack.pop();

				auto ret = builder.CreateTrunc(index, builder.getInt64Ty());
				ret = builder.CreateShl(ret, 32);
				size = builder.CreateTrunc(size, i32Ty);
				size = builder.CreateZExt(size, builder.getInt64Ty());
				ret = builder.CreateOr(ret, size);

				builder.CreateRet(ret);
				break;
			}

			case Instruction::SUICIDE:
			{
				auto address = stack.pop();
				ext.suicide(address);
				// Fall through
			}
			case Instruction::STOP:
			{
				builder.CreateRet(builder.getInt64(0));
				break;
			}

			}
		}

		if (!builder.GetInsertBlock()->getTerminator())	// If block not terminated
		{
			if (basicBlock.end() == bytecode.size())
			{
				//	Branch from the last regular block to the final block.
				builder.CreateBr(m_finalBlock->llvm());
			}
			else
			{
				// Branch to the next block.
				auto iterCopy = basicBlockPairIt;
				++iterCopy;
				auto& next = iterCopy->second;
				builder.CreateBr(next);
			}
		}
	}

	// Code for special blocks:
	builder.SetInsertPoint(m_finalBlock->llvm());
	builder.CreateRet(builder.getInt64(0));

	// TODO: throw an exception or something
	builder.SetInsertPoint(m_badJumpBlock->llvm());
	builder.CreateRet(builder.getInt64(1));

	linkBasicBlocks();

	return module;
}


void Compiler::linkBasicBlocks()
{
	/// Helper function that finds basic block given LLVM basic block pointer
	auto findBasicBlock = [this](llvm::BasicBlock* _llbb) -> BasicBlock&
	{
		// Name is used to get basic block index (index of first instruction)
		// TODO: If basicBlocs are still a map - multikey map can be used
		auto&& idxStr = _llbb->getName().substr(sizeof(BasicBlock::NamePrefix) - 2);
		auto idx = std::stoul(idxStr);
		return basicBlocks.find(idx)->second;
	};

	// Link basic blocks
	for (auto&& p : basicBlocks)
	{
		BasicBlock& bb = p.second;
		llvm::BasicBlock* llvmBB = bb.llvm();

		size_t valueIdx = 0;
		auto firstNonPhi = llvmBB->getFirstNonPHI();
		for (auto instIt = llvmBB->begin(); &*instIt != firstNonPhi; ++instIt, ++valueIdx)
		{
			auto phi = llvm::cast<llvm::PHINode>(instIt);
			for (auto predIt = llvm::pred_begin(llvmBB); predIt != llvm::pred_end(llvmBB); ++predIt)
			{
				auto& predBB = findBasicBlock(*predIt);
				assert(valueIdx < predBB.getStack().size()); // TODO: Report error
				phi->addIncoming(predBB.getStack().get(valueIdx), predBB);
			}
		}
	}
}

}
