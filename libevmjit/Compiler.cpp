
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

Compiler::Compiler(Options const& _options):
	m_options(_options),
	m_builder(llvm::getGlobalContext())
{
	Type::init(m_builder.getContext());
}

void Compiler::createBasicBlocks(bytesConstRef _bytecode)
{
	std::set<ProgramCounter> splitPoints; // Sorted collections of instruction indices where basic blocks start/end

	std::map<ProgramCounter, ProgramCounter> directJumpTargets;
	std::vector<ProgramCounter> indirectJumpTargets;
	boost::dynamic_bitset<> validJumpTargets(std::max(_bytecode.size(), size_t(1)));

	splitPoints.insert(0);  // First basic block
	validJumpTargets[0] = true;

	for (auto curr = _bytecode.begin(); curr != _bytecode.end(); ++curr)
	{
		ProgramCounter currentPC = curr - _bytecode.begin();
		validJumpTargets[currentPC] = true;

		auto inst = Instruction(*curr);
		switch (inst)
		{

		case Instruction::ANY_PUSH:
		{
			auto val = readPushData(curr, _bytecode.end());
			auto next = curr + 1;
			if (next == _bytecode.end())
				break;

			auto nextInst = Instruction(*next);
			if (nextInst == Instruction::JUMP || nextInst == Instruction::JUMPI)
			{
				// Create a block for the JUMP target.
				ProgramCounter targetPC = val < _bytecode.size() ? val.convert_to<ProgramCounter>() : _bytecode.size();
				splitPoints.insert(targetPC);

				ProgramCounter jumpPC = (next - _bytecode.begin());
				directJumpTargets[jumpPC] = targetPC;
			}
			break;
		}

		case Instruction::JUMPDEST:
		{
			// A basic block starts at the next instruction.
			if (currentPC + 1 < _bytecode.size())
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
			if (curr + 1 < _bytecode.end())
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
		if (*it > _bytecode.size() || !validJumpTargets[*it])
			it = splitPoints.erase(it);
		else
			++it;
	}

	for (auto it = splitPoints.cbegin(); it != splitPoints.cend();)
	{
		auto beginInstIdx = *it;
		++it;
		auto endInstIdx = it != splitPoints.cend() ? *it : _bytecode.size();
		basicBlocks.emplace(std::piecewise_construct, std::forward_as_tuple(beginInstIdx), std::forward_as_tuple(beginInstIdx, endInstIdx, m_mainFunc, m_builder));
	}

	m_stopBB = llvm::BasicBlock::Create(m_mainFunc->getContext(), "Stop", m_mainFunc);
	m_badJumpBlock = std::unique_ptr<BasicBlock>(new BasicBlock("BadJumpBlock", m_mainFunc, m_builder));
	m_jumpTableBlock = std::unique_ptr<BasicBlock>(new BasicBlock("JumpTableBlock", m_mainFunc, m_builder));

	for (auto it = directJumpTargets.cbegin(); it != directJumpTargets.cend(); ++it)
	{
		if (it->second >= _bytecode.size())
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
		m_indirectJumpTargets.push_back(&basicBlocks.find(*it)->second);
}

std::unique_ptr<llvm::Module> Compiler::compile(bytesConstRef _bytecode)
{
	auto module = std::unique_ptr<llvm::Module>(new llvm::Module("main", m_builder.getContext()));

	// Create main function
	llvm::Type* mainFuncArgTypes[] = {m_builder.getInt32Ty(), Type::RuntimePtr};    // There must be int in first place because LLVM does not support other signatures
	auto mainFuncType = llvm::FunctionType::get(Type::MainReturn, mainFuncArgTypes, false);
	m_mainFunc = llvm::Function::Create(mainFuncType, llvm::Function::ExternalLinkage, "main", module.get());
	m_mainFunc->arg_begin()->getNextNode()->setName("rt");

	// Create the basic blocks.
	auto entryBlock = llvm::BasicBlock::Create(m_builder.getContext(), "entry", m_mainFunc);
	m_builder.SetInsertPoint(entryBlock);

	createBasicBlocks(_bytecode);

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
		compileBasicBlock(basicBlock, _bytecode, runtimeManager, arith, memory, ext, gasMeter, nextBasicBlock);
	}

	// Code for special blocks:
	// TODO: move to separate function.
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
		m_builder.CreateBr(m_badJumpBlock->llvm());

	removeDeadBlocks();

	dumpCFGifRequired("blocks-init.dot");

	if (m_options.optimizeStack)
	{
		std::vector<BasicBlock*> blockList;
		for	(auto& entry : basicBlocks)
			blockList.push_back(&entry.second);

		if (m_jumpTableBlock)
			blockList.push_back(m_jumpTableBlock.get());

		BasicBlock::linkLocalStacks(blockList, m_builder);

		dumpCFGifRequired("blocks-opt.dot");
	}

	for (auto& entry : basicBlocks)
		entry.second.localStack().synchronize(stack);
	if (m_jumpTableBlock)
		m_jumpTableBlock->localStack().synchronize(stack);

	dumpCFGifRequired("blocks-sync.dot");

	if (m_jumpTableBlock && m_options.rewriteSwitchToBranches)
	{
		llvm::FunctionPassManager fpManager(module.get());
		fpManager.add(llvm::createLowerSwitchPass());
		fpManager.doInitialization();
		fpManager.run(*m_mainFunc);
	}

	return module;
}


void Compiler::compileBasicBlock(BasicBlock& _basicBlock, bytesConstRef _bytecode, RuntimeManager& _runtimeManager,
                                 Arith256& _arith, Memory& _memory, Ext& _ext, GasMeter& _gasMeter, llvm::BasicBlock* _nextBasicBlock)
{
	if (!_nextBasicBlock) // this is the last block in the code
		_nextBasicBlock = m_stopBB;

	m_builder.SetInsertPoint(_basicBlock.llvm());
	auto& stack = _basicBlock.localStack();

	for (auto currentPC = _basicBlock.begin(); currentPC != _basicBlock.end(); ++currentPC)
	{
		auto inst = static_cast<Instruction>(_bytecode[currentPC]);

		_gasMeter.count(inst);

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
			auto res = _arith.mul(lhs, rhs);
			stack.push(res);
			break;
		}

		case Instruction::DIV:
		{
			auto lhs = stack.pop();
			auto rhs = stack.pop();
			auto res = _arith.div(lhs, rhs);
			stack.push(res);
			break;
		}

		case Instruction::SDIV:
		{
			auto lhs = stack.pop();
			auto rhs = stack.pop();
			auto res = _arith.sdiv(lhs, rhs);
			stack.push(res);
			break;
		}

		case Instruction::MOD:
		{
			auto lhs = stack.pop();
			auto rhs = stack.pop();
			auto res = _arith.mod(lhs, rhs);
			stack.push(res);
			break;
		}

		case Instruction::SMOD:
		{
			auto lhs = stack.pop();
			auto rhs = stack.pop();
			auto res = _arith.smod(lhs, rhs);
			stack.push(res);
			break;
		}

		case Instruction::EXP:
		{
			auto left = stack.pop();
			auto right = stack.pop();
			auto ret = _ext.exp(left, right);
			stack.push(ret);
			break;
		}

		case Instruction::BNOT:
		{
			auto value = stack.pop();
			auto ret = m_builder.CreateXor(value, llvm::APInt(256, -1, true), "bnot");
			stack.push(ret);
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
			auto res = _arith.addmod(lhs, rhs, mod);
			stack.push(res);
			break;
		}

		case Instruction::MULMOD:
		{
			auto lhs = stack.pop();
			auto rhs = stack.pop();
			auto mod = stack.pop();
			auto res = _arith.mulmod(lhs, rhs, mod);
			stack.push(res);
			break;
		}

		case Instruction::SIGNEXTEND:
		{
			auto idx = stack.pop();
			auto word = stack.pop();

			auto k32_ = m_builder.CreateTrunc(idx, m_builder.getIntNTy(5), "k_32");
			auto k32 = m_builder.CreateZExt(k32_, Type::i256);
			auto k32x8 = m_builder.CreateMul(k32, Constant::get(8), "kx8");

			// test for word >> (k * 8 + 7)
			auto bitpos = m_builder.CreateAdd(k32x8, Constant::get(7), "bitpos");
			auto bitval = m_builder.CreateLShr(word, bitpos, "bitval");
			auto bittest = m_builder.CreateTrunc(bitval, m_builder.getInt1Ty(), "bittest");

			auto mask_ = m_builder.CreateShl(Constant::get(1), bitpos);
			auto mask = m_builder.CreateSub(mask_, Constant::get(1), "mask");

			auto negmask = m_builder.CreateXor(mask, llvm::ConstantInt::getAllOnesValue(Type::i256), "negmask");
			auto val1 = m_builder.CreateOr(word, negmask);
			auto val0 = m_builder.CreateAnd(word, mask);

			auto kInRange = m_builder.CreateICmpULE(idx, llvm::ConstantInt::get(Type::i256, 30));
			auto result = m_builder.CreateSelect(kInRange,
			                                     m_builder.CreateSelect(bittest, val1, val0),
			                                     word);
			stack.push(result);

			break;
		}

		case Instruction::SHA3:
		{
			auto inOff = stack.pop();
			auto inSize = stack.pop();
			_memory.require(inOff, inSize);
			auto hash = _ext.sha3(inOff, inSize);
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
			auto curr = _bytecode.begin() + currentPC;	// TODO: replace currentPC with iterator
			auto value = readPushData(curr, _bytecode.end());
			currentPC = curr - _bytecode.begin();

			stack.push(Constant::get(value));
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
			auto word = _memory.loadWord(addr);
			stack.push(word);
			break;
		}

		case Instruction::MSTORE:
		{
			auto addr = stack.pop();
			auto word = stack.pop();
			_memory.storeWord(addr, word);
			break;
		}

		case Instruction::MSTORE8:
		{
			auto addr = stack.pop();
			auto word = stack.pop();
			_memory.storeByte(addr, word);
			break;
		}

		case Instruction::MSIZE:
		{
			auto word = _memory.getSize();
			stack.push(word);
			break;
		}

		case Instruction::SLOAD:
		{
			auto index = stack.pop();
			auto value = _ext.store(index);
			stack.push(value);
			break;
		}

		case Instruction::SSTORE:
		{
			auto index = stack.pop();
			auto value = stack.pop();
			_gasMeter.countSStore(_ext, index, value);
			_ext.setStore(index, value);
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
			if (currentPC != _basicBlock.begin())
			{
				auto pairIter = m_directJumpTargets.find(currentPC);
				if (pairIter != m_directJumpTargets.end())
					targetBlock = pairIter->second;
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
					m_builder.CreateBr(m_jumpTableBlock->llvm());
			}
			else // JUMPI
			{
				stack.swap(1);
				auto val = stack.pop();
				auto zero = Constant::get(0);
				auto cond = m_builder.CreateICmpNE(val, zero, "nonzero");

				if (targetBlock)
				{
					stack.pop();
					m_builder.CreateCondBr(cond, targetBlock, _nextBasicBlock);
				}
				else
					m_builder.CreateCondBr(cond, m_jumpTableBlock->llvm(), _nextBasicBlock);
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
			auto value = _ext.balance(address);
			stack.push(value);
			break;
		}

		case Instruction::EXTCODESIZE:
		{
			auto addr = stack.pop();
			auto value = _ext.codesizeAt(addr);
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

			_memory.copyBytes(srcPtr, srcSize, srcIdx, destMemIdx, reqBytes);
			break;
		}

		case Instruction::CODECOPY:
		{
			auto destMemIdx = stack.pop();
			auto srcIdx = stack.pop();
			auto reqBytes = stack.pop();

			auto srcPtr = _runtimeManager.getCode();    // TODO: Code & its size are constants, feature #80814234
			auto srcSize = _runtimeManager.get(RuntimeData::CodeSize);

			_memory.copyBytes(srcPtr, srcSize, srcIdx, destMemIdx, reqBytes);
			break;
		}

		case Instruction::EXTCODECOPY:
		{
			auto extAddr = stack.pop();
			auto destMemIdx = stack.pop();
			auto srcIdx = stack.pop();
			auto reqBytes = stack.pop();

			auto srcPtr = _ext.codeAt(extAddr);
			auto srcSize = _ext.codesizeAt(extAddr);

			_memory.copyBytes(srcPtr, srcSize, srcIdx, destMemIdx, reqBytes);
			break;
		}

		case Instruction::CALLDATALOAD:
		{
			auto index = stack.pop();
			auto value = _ext.calldataload(index);
			stack.push(value);
			break;
		}

		case Instruction::CREATE:
		{
			auto endowment = stack.pop();
			auto initOff = stack.pop();
			auto initSize = stack.pop();
			_memory.require(initOff, initSize);

			auto address = _ext.create(endowment, initOff, initSize);
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

			_gasMeter.commitCostBlock(gas);

			// Require _memory for the max of in and out buffers
			auto inSizeReq = m_builder.CreateAdd(inOff, inSize, "inSizeReq");
			auto outSizeReq = m_builder.CreateAdd(outOff, outSize, "outSizeReq");
			auto cmp = m_builder.CreateICmpUGT(inSizeReq, outSizeReq);
			auto sizeReq = m_builder.CreateSelect(cmp, inSizeReq, outSizeReq, "sizeReq");
			_memory.require(sizeReq);

			auto receiveAddress = codeAddress;
			if (inst == Instruction::CALLCODE)
				receiveAddress = _runtimeManager.get(RuntimeData::Address);

			auto ret = _ext.call(gas, receiveAddress, value, inOff, inSize, outOff, outSize, codeAddress);
			_gasMeter.giveBack(gas);
			stack.push(ret);
			break;
		}

		case Instruction::RETURN:
		{
			auto index = stack.pop();
			auto size = stack.pop();

			_memory.require(index, size);
			_runtimeManager.registerReturnData(index, size);

			m_builder.CreateRet(Constant::get(ReturnCode::Return));
			break;
		}

		case Instruction::SUICIDE:
		case Instruction::STOP:
		{
			if (inst == Instruction::SUICIDE)
			{
				auto address = stack.pop();
				_ext.suicide(address);
			}

			m_builder.CreateRet(Constant::get(ReturnCode::Stop));
			break;
		}

		default: // Invalid instruction - runtime exception
		{
			_runtimeManager.raiseException(ReturnCode::BadInstruction);
		}

		}
	}

	_gasMeter.commitCostBlock();

	// Block may have no terminator if the next instruction is a jump destination.
	if (!_basicBlock.llvm()->getTerminator())
		m_builder.CreateBr(_nextBasicBlock);
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

void Compiler::dumpCFGifRequired(std::string const& _dotfilePath)
{
	if (! m_options.dumpCFG)
		return;

	// TODO: handle i/o failures
	std::ofstream ofs(_dotfilePath);
	dumpCFGtoStream(ofs);
	ofs.close();
}

void Compiler::dumpCFGtoStream(std::ostream& _out)
{
	_out << "digraph BB {\n"
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

		_out << " \"" << blockName << "\" [shape=record, label=\" { " << blockName << "|" << oss.str() << "} \"];\n";
	}

	// Output edges
	for (auto bb : blocks)
	{
		std::string blockName = bb->llvm()->getName();

		auto end = llvm::pred_end(bb->llvm());
		for (llvm::pred_iterator it = llvm::pred_begin(bb->llvm()); it != end; ++it)
		{
			_out << "  \"" << (*it)->getName().str() << "\" -> \"" << blockName << "\" ["
				 << ((m_jumpTableBlock.get() && *it == m_jumpTableBlock.get()->llvm()) ? "style = dashed, " : "")
				 << "];\n";
		}
	}

	_out << "}\n";
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

