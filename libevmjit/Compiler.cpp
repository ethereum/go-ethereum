
#include "Compiler.h"

#include <fstream>

#include <boost/dynamic_bitset.hpp>

#include <llvm/ADT/PostOrderIterator.h>
#include <llvm/IR/CFG.h>
#include <llvm/IR/Module.h>
#include <llvm/IR/IntrinsicInst.h>

#include <llvm/PassManager.h>
#include <llvm/Transforms/Scalar.h>

#include <libevmface/Instruction.h>

#include "Type.h"
#include "Memory.h"
#include "Stack.h"
#include "Ext.h"
#include "GasMeter.h"
#include "Utils.h"
#include "Endianness.h"
#include "Arith256.h"
#include "Runtime.h"

namespace dev
{
namespace eth
{
namespace jit
{

Compiler::Compiler():
	m_builder(llvm::getGlobalContext())
{
	Type::init(m_builder.getContext());
}

void Compiler::createBasicBlocks(bytesConstRef bytecode)
{
	std::set<ProgramCounter> splitPoints; // Sorted collections of instruction indices where basic blocks start/end

	std::map<ProgramCounter, ProgramCounter> directJumpTargets;
	std::vector<ProgramCounter> indirectJumpTargets;
	boost::dynamic_bitset<> validJumpTargets(std::max(bytecode.size(), size_t(1)));

	splitPoints.insert(0);  // First basic block
	validJumpTargets[0] = true;

	for (auto curr = bytecode.begin(); curr != bytecode.end(); ++curr)
	{
		ProgramCounter currentPC = curr - bytecode.begin();
		validJumpTargets[currentPC] = true;

		auto inst = static_cast<Instruction>(*curr);
		switch (inst)
		{

		case Instruction::ANY_PUSH:
		{
			auto numBytes = static_cast<size_t>(inst) - static_cast<size_t>(Instruction::PUSH1) + 1;
			auto next = curr + numBytes + 1;
			if (next >= bytecode.end())
				break;

			auto nextInst = static_cast<Instruction>(*next);

			if (nextInst == Instruction::JUMP || nextInst == Instruction::JUMPI)
			{
				// Compute target PC of the jump.
				u256 val = 0;
				for (auto iter = curr + 1; iter < next; ++iter)
				{
					val <<= 8;
					val |= *iter;
				}

				// Create a block for the JUMP target.
				ProgramCounter targetPC = val < bytecode.size() ? val.convert_to<ProgramCounter>() : bytecode.size();
				splitPoints.insert(targetPC);

				ProgramCounter jumpPC = (next - bytecode.begin());
				directJumpTargets[jumpPC] = targetPC;
			}

			curr += numBytes;
			break;
		}

		case Instruction::JUMPDEST:
		{
			// A basic block starts at the next instruction.
			if (currentPC + 1 < bytecode.size())
			{
				splitPoints.insert(currentPC + 1);
				indirectJumpTargets.push_back(currentPC + 1);
			}
			break;
		}

		case Instruction::JUMP:
		case Instruction::JUMPI:
		case Instruction::RETURN:
		case Instruction::STOP:
		case Instruction::SUICIDE:
		{
			// Create a basic block starting at the following instruction.
			if (curr + 1 < bytecode.end())
			{
				splitPoints.insert(currentPC + 1);
			}
			break;
		}

		default:
			break;
		}
	}

	// Remove split points generated from jumps out of code or into data.
	for (auto it = splitPoints.cbegin(); it != splitPoints.cend();)
	{
		if (*it > bytecode.size() || !validJumpTargets[*it])
			it = splitPoints.erase(it);
		else
			++it;
	}

	for (auto it = splitPoints.cbegin(); it != splitPoints.cend();)
	{
		auto beginInstIdx = *it;
		++it;
		auto endInstIdx = it != splitPoints.cend() ? *it : bytecode.size();
		basicBlocks.emplace(std::piecewise_construct, std::forward_as_tuple(beginInstIdx), std::forward_as_tuple(beginInstIdx, endInstIdx, m_mainFunc, m_builder));
	}

	m_stopBB = llvm::BasicBlock::Create(m_mainFunc->getContext(), "Stop", m_mainFunc);
	m_badJumpBlock = std::make_unique<BasicBlock>("BadJumpBlock", m_mainFunc, m_builder);
	m_jumpTableBlock = std::make_unique<BasicBlock>("JumpTableBlock", m_mainFunc, m_builder);

	for (auto it = directJumpTargets.cbegin(); it != directJumpTargets.cend(); ++it)
	{
		if (it->second >= bytecode.size())
		{
			// Jumping out of code means STOP
			m_directJumpTargets[it->first] = m_stopBB;
			continue;
		}

		auto blockIter = basicBlocks.find(it->second);
		if (blockIter != basicBlocks.end())
		{
			m_directJumpTargets[it->first] = blockIter->second.llvm();
		}
		else
		{
			clog(JIT) << "Bad JUMP at PC " << it->first
					  << ": " << it->second << " is not a valid PC";
			m_directJumpTargets[it->first] = m_badJumpBlock->llvm();
		}
	}

	for (auto it = indirectJumpTargets.cbegin(); it != indirectJumpTargets.cend(); ++it)
	{
		m_indirectJumpTargets.push_back(&basicBlocks.find(*it)->second);
	}
}

std::unique_ptr<llvm::Module> Compiler::compile(bytesConstRef bytecode)
{
	auto module = std::make_unique<llvm::Module>("main", m_builder.getContext());

	// Create main function
	llvm::Type* mainFuncArgTypes[] = {m_builder.getInt32Ty(), Type::RuntimePtr};    // There must be int in first place because LLVM does not support other signatures
	auto mainFuncType = llvm::FunctionType::get(Type::MainReturn, mainFuncArgTypes, false);
	m_mainFunc = llvm::Function::Create(mainFuncType, llvm::Function::ExternalLinkage, "main", module.get());
	m_mainFunc->arg_begin()->getNextNode()->setName("rt");

	// Create the basic blocks.
	auto entryBlock = llvm::BasicBlock::Create(m_builder.getContext(), "entry", m_mainFunc);
	m_builder.SetInsertPoint(entryBlock);

	createBasicBlocks(bytecode);

	// Init runtime structures.
	RuntimeManager runtimeManager(m_builder);
	GasMeter gasMeter(m_builder, runtimeManager);
	Memory memory(runtimeManager, gasMeter);
	Ext ext(runtimeManager);
	Stack stack(m_builder, runtimeManager);
	Arith256 arith(m_builder);

	m_builder.CreateBr(basicBlocks.begin()->second);

	for (auto basicBlockPairIt = basicBlocks.begin(); basicBlockPairIt != basicBlocks.end(); ++basicBlockPairIt)
	{
		auto& basicBlock = basicBlockPairIt->second;
		auto iterCopy = basicBlockPairIt;
		++iterCopy;
		auto nextBasicBlock = (iterCopy != basicBlocks.end()) ? iterCopy->second.llvm() : nullptr;
		compileBasicBlock(basicBlock, bytecode, runtimeManager, arith, memory, ext, gasMeter, nextBasicBlock);
	}

	// Code for special blocks:
	// TODO: move to separate function.
	// Note: Right now the codegen for special blocks depends only on createBasicBlock(),
	// not on the codegen for 'regular' blocks. But it has to be done before linkBasicBlocks().
	m_builder.SetInsertPoint(m_stopBB);
	m_builder.CreateRet(Constant::get(ReturnCode::Stop));

	m_builder.SetInsertPoint(m_badJumpBlock->llvm());
	m_builder.CreateRet(Constant::get(ReturnCode::BadJumpDestination));

	m_builder.SetInsertPoint(m_jumpTableBlock->llvm());
	if (m_indirectJumpTargets.size() > 0)
	{
		auto dest = m_jumpTableBlock->localStack().pop();
		auto switchInstr =  m_builder.CreateSwitch(dest, m_badJumpBlock->llvm(),
							m_indirectJumpTargets.size());
		for (auto it = m_indirectJumpTargets.cbegin(); it != m_indirectJumpTargets.cend(); ++it)
		{
			auto& bb = *it;
			auto dest = Constant::get(bb->begin());
			switchInstr->addCase(dest, bb->llvm());
		}
	}
	else
	{
		m_builder.CreateBr(m_badJumpBlock->llvm());
	}

	removeDeadBlocks();

	if (getenv("EVMCC_DEBUG_BLOCKS"))
	{
		std::ofstream ofs("blocks-init.dot");
		dumpBasicBlockGraph(ofs);
		ofs.close();
		std::cerr << "\n\nAfter dead block elimination \n\n";
		dump();
	}

	//if (getenv("EVMCC_OPTIMIZE_STACK"))
	//{
		std::vector<BasicBlock*> blockList;
		for	(auto& entry : basicBlocks)
			blockList.push_back(&entry.second);

		if (m_jumpTableBlock != nullptr)
			blockList.push_back(m_jumpTableBlock.get());

		BasicBlock::linkLocalStacks(blockList, m_builder);

		if (getenv("EVMCC_DEBUG_BLOCKS"))
		{
			std::ofstream ofs("blocks-opt.dot");
			dumpBasicBlockGraph(ofs);
			ofs.close();
			std::cerr << "\n\nAfter stack optimization \n\n";
			dump();
		}
	//}

	for (auto& entry : basicBlocks)
		entry.second.localStack().synchronize(stack);
	if (m_jumpTableBlock != nullptr)
		m_jumpTableBlock->localStack().synchronize(stack);

	if (getenv("EVMCC_DEBUG_BLOCKS"))
	{
		std::ofstream ofs("blocks-sync.dot");
		dumpBasicBlockGraph(ofs);
		ofs.close();
		std::cerr << "\n\nAfter stack synchronization \n\n";
		dump();
	}

	llvm::FunctionPassManager fpManager(module.get());
	fpManager.add(llvm::createLowerSwitchPass());
	fpManager.doInitialization();
	fpManager.run(*m_mainFunc);

	return module;
}


void Compiler::compileBasicBlock(BasicBlock& basicBlock, bytesConstRef bytecode, RuntimeManager& _runtimeManager, Arith256& arith, Memory& memory, Ext& ext, GasMeter& gasMeter, llvm::BasicBlock* nextBasicBlock)
{
	m_builder.SetInsertPoint(basicBlock.llvm());
	auto& stack = basicBlock.localStack();

	for (auto currentPC = basicBlock.begin(); currentPC != basicBlock.end(); ++currentPC)
	{
		auto inst = static_cast<Instruction>(bytecode[currentPC]);

		gasMeter.count(inst);

		switch (inst)
		{

		case Instruction::ADD:
		{
			auto lhs = stack.pop();
			auto rhs = stack.pop();
			auto result = m_builder.CreateAdd(lhs, rhs);
			stack.push(result);
			break;
		}

		case Instruction::SUB:
		{
			auto lhs = stack.pop();
			auto rhs = stack.pop();
			auto result = m_builder.CreateSub(lhs, rhs);
			stack.push(result);
			break;
		}

		case Instruction::MUL:
		{
			auto lhs = stack.pop();
			auto rhs = stack.pop();
			auto res = arith.mul(lhs, rhs);
			stack.push(res);
			break;
		}

		case Instruction::DIV:
		{
			auto lhs = stack.pop();
			auto rhs = stack.pop();
			auto res = arith.div(lhs, rhs);
			stack.push(res);
			break;
		}

		case Instruction::SDIV:
		{
			auto lhs = stack.pop();
			auto rhs = stack.pop();
			auto res = arith.sdiv(lhs, rhs);
			stack.push(res);
			break;
		}

		case Instruction::MOD:
		{
			auto lhs = stack.pop();
			auto rhs = stack.pop();
			auto res = arith.mod(lhs, rhs);
			stack.push(res);
			break;
		}

		case Instruction::SMOD:
		{
			auto lhs = stack.pop();
			auto rhs = stack.pop();
			auto res = arith.smod(lhs, rhs);
			stack.push(res);
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

		case Instruction::BNOT:
		{
			auto top = stack.pop();
			auto allones = llvm::ConstantInt::get(Type::i256, llvm::APInt::getAllOnesValue(256));
			auto res = m_builder.CreateXor(top, allones);
			stack.push(res);
			break;
		}

		case Instruction::LT:
		{
			auto lhs = stack.pop();
			auto rhs = stack.pop();
			auto res1 = m_builder.CreateICmpULT(lhs, rhs);
			auto res256 = m_builder.CreateZExt(res1, Type::i256);
			stack.push(res256);
			break;
		}

		case Instruction::GT:
		{
			auto lhs = stack.pop();
			auto rhs = stack.pop();
			auto res1 = m_builder.CreateICmpUGT(lhs, rhs);
			auto res256 = m_builder.CreateZExt(res1, Type::i256);
			stack.push(res256);
			break;
		}

		case Instruction::SLT:
		{
			auto lhs = stack.pop();
			auto rhs = stack.pop();
			auto res1 = m_builder.CreateICmpSLT(lhs, rhs);
			auto res256 = m_builder.CreateZExt(res1, Type::i256);
			stack.push(res256);
			break;
		}

		case Instruction::SGT:
		{
			auto lhs = stack.pop();
			auto rhs = stack.pop();
			auto res1 = m_builder.CreateICmpSGT(lhs, rhs);
			auto res256 = m_builder.CreateZExt(res1, Type::i256);
			stack.push(res256);
			break;
		}

		case Instruction::EQ:
		{
			auto lhs = stack.pop();
			auto rhs = stack.pop();
			auto res1 = m_builder.CreateICmpEQ(lhs, rhs);
			auto res256 = m_builder.CreateZExt(res1, Type::i256);
			stack.push(res256);
			break;
		}

		case Instruction::NOT:
		{
			auto top = stack.pop();
			auto iszero = m_builder.CreateICmpEQ(top, Constant::get(0), "iszero");
			auto result = m_builder.CreateZExt(iszero, Type::i256);
			stack.push(result);
			break;
		}

		case Instruction::AND:
		{
			auto lhs = stack.pop();
			auto rhs = stack.pop();
			auto res = m_builder.CreateAnd(lhs, rhs);
			stack.push(res);
			break;
		}

		case Instruction::OR:
		{
			auto lhs = stack.pop();
			auto rhs = stack.pop();
			auto res = m_builder.CreateOr(lhs, rhs);
			stack.push(res);
			break;
		}

		case Instruction::XOR:
		{
			auto lhs = stack.pop();
			auto rhs = stack.pop();
			auto res = m_builder.CreateXor(lhs, rhs);
			stack.push(res);
			break;
		}

		case Instruction::BYTE:
		{
			const auto byteNum = stack.pop();
			auto value = stack.pop();

			//
			value = Endianness::toBE(m_builder, value);
			auto bytes = m_builder.CreateBitCast(value, llvm::VectorType::get(Type::Byte, 32), "bytes");
			auto byte = m_builder.CreateExtractElement(bytes, byteNum, "byte");
			value = m_builder.CreateZExt(byte, Type::i256);

			auto byteNumValid = m_builder.CreateICmpULT(byteNum, Constant::get(32));
			value = m_builder.CreateSelect(byteNumValid, value, Constant::get(0));
			stack.push(value);

			break;
		}

		case Instruction::ADDMOD:
		{
			auto lhs = stack.pop();
			auto rhs = stack.pop();
			auto mod = stack.pop();
			auto res = arith.addmod(lhs, rhs, mod);
			stack.push(res);
			break;
		}

		case Instruction::MULMOD:
		{
			auto lhs = stack.pop();
			auto rhs = stack.pop();
			auto mod = stack.pop();
			auto res = arith.mulmod(lhs, rhs, mod);
			stack.push(res);
			break;
		}

		case Instruction::SIGNEXTEND:
		{
			auto k = stack.pop();
			auto b = stack.pop();
			auto k32 = m_builder.CreateTrunc(k, m_builder.getIntNTy(5), "k_32");
			auto k32ext = m_builder.CreateZExt(k32, Type::i256);
			auto k32x8 = m_builder.CreateMul(k32ext, Constant::get(8), "kx8");

			// test for b >> (k * 8 + 7)
			auto val = m_builder.CreateAdd(k32x8, llvm::ConstantInt::get(Type::i256, 7));
			auto tmp = m_builder.CreateAShr(b, val);
			auto bitset = m_builder.CreateTrunc(tmp, m_builder.getInt1Ty());

			// shift left by (31 - k) * 8 = (248 - k*8), then do arithmetic shr by the same amount.
			auto shiftSize = m_builder.CreateSub(llvm::ConstantInt::get(Type::i256, 31 * 8), k32x8);
			auto bshl = m_builder.CreateShl(b, shiftSize);
			auto bshr = m_builder.CreateAShr(bshl, shiftSize);

			// bshr is our final value if 0 <= k <= 30 and bitset is true,
			// otherwise we push back b unchanged
			auto kInRange = m_builder.CreateICmpULE(k, llvm::ConstantInt::get(Type::i256, 30));
			auto cond = m_builder.CreateAnd(kInRange, bitset);

			auto result = m_builder.CreateSelect(cond, bshr, b);
			stack.push(result);

			break;
		}

		case Instruction::SHA3:
		{
			auto inOff = stack.pop();
			auto inSize = stack.pop();
			memory.require(inOff, inSize);
			auto hash = ext.sha3(inOff, inSize);
			stack.push(hash);
			break;
		}

		case Instruction::POP:
		{
			stack.pop();
			break;
		}

		case Instruction::ANY_PUSH:
		{
			auto numBytes = static_cast<size_t>(inst) - static_cast<size_t>(Instruction::PUSH1) + 1;
			auto value = llvm::APInt(256, 0);
			for (decltype(numBytes) i = 0; i < numBytes; ++i)   // TODO: Use pc as iterator
			{
				++currentPC;
				value <<= 8;
				value |= bytecode[currentPC];
			}
			auto c = m_builder.getInt(value);
			stack.push(c);
			break;
		}

		case Instruction::ANY_DUP:
		{
			auto index = static_cast<size_t>(inst) - static_cast<size_t>(Instruction::DUP1);
			stack.dup(index);
			break;
		}

		case Instruction::ANY_SWAP:
		{
			auto index = static_cast<size_t>(inst) - static_cast<size_t>(Instruction::SWAP1) + 1;
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
			gasMeter.countSStore(ext, index, value);
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
			llvm::BasicBlock* targetBlock = nullptr;
			if (currentPC != basicBlock.begin())
			{
				auto pairIter = m_directJumpTargets.find(currentPC);
				if (pairIter != m_directJumpTargets.end())
				{
					targetBlock = pairIter->second;
				}
			}

			if (inst == Instruction::JUMP)
			{
				if (targetBlock)
				{
					// The target address is computed at compile time,
					// just pop it without looking...
					stack.pop();
					m_builder.CreateBr(targetBlock);
				}
				else
				{
					m_builder.CreateBr(m_jumpTableBlock->llvm());
				}
			}
			else // JUMPI
			{
				stack.swap(1);
				auto val = stack.pop();
				auto zero = Constant::get(0);
				auto cond = m_builder.CreateICmpNE(val, zero, "nonzero");

				// Assume the basic blocks are properly ordered:
				assert(nextBasicBlock); // FIXME: JUMPI can be last instruction

				if (targetBlock)
				{
					stack.pop();
					m_builder.CreateCondBr(cond, targetBlock, nextBasicBlock);
				}
				else
				{
					m_builder.CreateCondBr(cond, m_jumpTableBlock->llvm(), nextBasicBlock);
				}
			}

			break;
		}

		case Instruction::JUMPDEST:
		{
			// Nothing to do
			break;
		}

		case Instruction::PC:
		{
			auto value = Constant::get(currentPC);
			stack.push(value);
			break;
		}

		case Instruction::GAS:
		case Instruction::ADDRESS:
		case Instruction::CALLER:
		case Instruction::ORIGIN:
		case Instruction::CALLVALUE:
		case Instruction::CALLDATASIZE:
		case Instruction::CODESIZE:
		case Instruction::GASPRICE:
		case Instruction::PREVHASH:
		case Instruction::COINBASE:
		case Instruction::TIMESTAMP:
		case Instruction::NUMBER:
		case Instruction::DIFFICULTY:
		case Instruction::GASLIMIT:
		{
			// Pushes an element of runtime data on stack
			stack.push(_runtimeManager.get(inst));
			break;
		}

		case Instruction::BALANCE:
		{
			auto address = stack.pop();
			auto value = ext.balance(address);
			stack.push(value);
			break;
		}

		case Instruction::EXTCODESIZE:
		{
			auto addr = stack.pop();
			auto value = ext.codesizeAt(addr);
			stack.push(value);
			break;
		}

		case Instruction::CALLDATACOPY:
		{
			auto destMemIdx = stack.pop();
			auto srcIdx = stack.pop();
			auto reqBytes = stack.pop();

			auto srcPtr = _runtimeManager.getCallData();
			auto srcSize = _runtimeManager.get(RuntimeData::CallDataSize);

			memory.copyBytes(srcPtr, srcSize, srcIdx, destMemIdx, reqBytes);
			break;
		}

		case Instruction::CODECOPY:
		{
			auto destMemIdx = stack.pop();
			auto srcIdx = stack.pop();
			auto reqBytes = stack.pop();

			auto srcPtr = _runtimeManager.getCode();    // TODO: Code & its size are constants, feature #80814234
			auto srcSize = _runtimeManager.get(RuntimeData::CodeSize);

			memory.copyBytes(srcPtr, srcSize, srcIdx, destMemIdx, reqBytes);
			break;
		}

		case Instruction::EXTCODECOPY:
		{
			auto extAddr = stack.pop();
			auto destMemIdx = stack.pop();
			auto srcIdx = stack.pop();
			auto reqBytes = stack.pop();

			auto srcPtr = ext.codeAt(extAddr);
			auto srcSize = ext.codesizeAt(extAddr);

			memory.copyBytes(srcPtr, srcSize, srcIdx, destMemIdx, reqBytes);
			break;
		}

		case Instruction::CALLDATALOAD:
		{
			auto index = stack.pop();
			auto value = ext.calldataload(index);
			stack.push(value);
			break;
		}

		case Instruction::CREATE:
		{
			auto endowment = stack.pop();
			auto initOff = stack.pop();
			auto initSize = stack.pop();
			memory.require(initOff, initSize);

			auto address = ext.create(endowment, initOff, initSize);
			stack.push(address);
			break;
		}

		case Instruction::CALL:
		case Instruction::CALLCODE:
		{
			auto gas = stack.pop();
			auto codeAddress = stack.pop();
			auto value = stack.pop();
			auto inOff = stack.pop();
			auto inSize = stack.pop();
			auto outOff = stack.pop();
			auto outSize = stack.pop();

			gasMeter.commitCostBlock(gas);

			// Require memory for the max of in and out buffers
			auto inSizeReq = m_builder.CreateAdd(inOff, inSize, "inSizeReq");
			auto outSizeReq = m_builder.CreateAdd(outOff, outSize, "outSizeReq");
			auto cmp = m_builder.CreateICmpUGT(inSizeReq, outSizeReq);
			auto sizeReq = m_builder.CreateSelect(cmp, inSizeReq, outSizeReq, "sizeReq");
			memory.require(sizeReq);

			auto receiveAddress = codeAddress;
			if (inst == Instruction::CALLCODE)
				receiveAddress = _runtimeManager.get(RuntimeData::Address);

			auto ret = ext.call(gas, receiveAddress, value, inOff, inSize, outOff, outSize, codeAddress);
			gasMeter.giveBack(gas);
			stack.push(ret);
			break;
		}

		case Instruction::RETURN:
		{
			auto index = stack.pop();
			auto size = stack.pop();

			memory.require(index, size);
			_runtimeManager.registerReturnData(index, size);

			m_builder.CreateRet(Constant::get(ReturnCode::Return));
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
			m_builder.CreateRet(Constant::get(ReturnCode::Stop));
			break;
		}

		default: // Invalid instruction - runtime exception
		{
			_runtimeManager.raiseException(ReturnCode::BadInstruction);
		}

		}
	}

	gasMeter.commitCostBlock();

	if (!basicBlock.llvm()->getTerminator())    // If block not terminated
	{
		if (nextBasicBlock)
			m_builder.CreateBr(nextBasicBlock); // Branch to the next block
		else
			m_builder.CreateRet(Constant::get(ReturnCode::Stop));   // Return STOP code
	}
}



void Compiler::removeDeadBlocks()
{
	// Remove dead basic blocks
	auto sthErased = false;
	do
	{
		sthErased = false;
		for (auto it = basicBlocks.begin(); it != basicBlocks.end();)
		{
			auto llvmBB = it->second.llvm();
			if (llvm::pred_begin(llvmBB) == llvm::pred_end(llvmBB))
			{
				llvmBB->eraseFromParent();
				basicBlocks.erase(it++);
				sthErased = true;
			}
			else
				++it;
		}
	}
	while (sthErased);

	// Remove jump table block if no predecessors
	if (llvm::pred_begin(m_jumpTableBlock->llvm()) == llvm::pred_end(m_jumpTableBlock->llvm()))
	{
		m_jumpTableBlock->llvm()->eraseFromParent();
		m_jumpTableBlock.reset();
	}
}

void Compiler::dumpBasicBlockGraph(std::ostream& out)
{
	out << "digraph BB {\n"
		<< "  node [shape=record, fontname=Courier, fontsize=10];\n"
		<< "  entry [share=record, label=\"entry block\"];\n";

	std::vector<BasicBlock*> blocks;
	for (auto& pair : basicBlocks)
		blocks.push_back(&pair.second);
	if (m_jumpTableBlock)
		blocks.push_back(m_jumpTableBlock.get());
	if (m_badJumpBlock)
		blocks.push_back(m_badJumpBlock.get());

	// std::map<BasicBlock*,int> phiNodesPerBlock;

	// Output nodes
	for (auto bb : blocks)
	{
		std::string blockName = bb->llvm()->getName();

		std::ostringstream oss;
		bb->dump(oss, true);

		out << " \"" << blockName << "\" [shape=record, label=\" { " << blockName << "|" << oss.str() << "} \"];\n";
	}

	// Output edges
	for (auto bb : blocks)
	{
		std::string blockName = bb->llvm()->getName();

		auto end = llvm::pred_end(bb->llvm());
		for (llvm::pred_iterator it = llvm::pred_begin(bb->llvm()); it != end; ++it)
		{
			out << "  \"" << (*it)->getName().str() << "\" -> \"" << blockName << "\" ["
				<< ((m_jumpTableBlock.get() && *it == m_jumpTableBlock.get()->llvm()) ? "style = dashed, " : "")
				//<< "label = \""
				//<< phiNodesPerBlock[bb]
				<< "];\n";
		}
	}

	out << "}\n";
}

void Compiler::dump()
{
	for (auto& entry : basicBlocks)
		entry.second.dump();
	if (m_jumpTableBlock != nullptr)
		m_jumpTableBlock->dump();
}

}
}
}

