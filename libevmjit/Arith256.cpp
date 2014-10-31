#include "Arith256.h"
#include "Runtime.h"
#include "Type.h"

#include <llvm/IR/Function.h>

#include <libdevcore/Common.h>

namespace dev
{
namespace eth
{
namespace jit
{

Arith256::Arith256(llvm::IRBuilder<>& _builder) :
	CompilerHelper(_builder)
{
	using namespace llvm;

	m_result = m_builder.CreateAlloca(Type::Word, nullptr, "arith.result");
	m_arg1 = m_builder.CreateAlloca(Type::Word, nullptr, "arith.arg1");
	m_arg2 = m_builder.CreateAlloca(Type::Word, nullptr, "arith.arg2");
	m_arg3 = m_builder.CreateAlloca(Type::Word, nullptr, "arith.arg3");

	using Linkage = GlobalValue::LinkageTypes;

	llvm::Type* arg2Types[] = {Type::WordPtr, Type::WordPtr, Type::WordPtr};
	llvm::Type* arg3Types[] = {Type::WordPtr, Type::WordPtr, Type::WordPtr, Type::WordPtr};

	m_mul = Function::Create(FunctionType::get(Type::Void, arg2Types, false), Linkage::ExternalLinkage, "arith_mul", getModule());
	m_div = Function::Create(FunctionType::get(Type::Void, arg2Types, false), Linkage::ExternalLinkage, "arith_div", getModule());
	m_mod = Function::Create(FunctionType::get(Type::Void, arg2Types, false), Linkage::ExternalLinkage, "arith_mod", getModule());
	m_sdiv = Function::Create(FunctionType::get(Type::Void, arg2Types, false), Linkage::ExternalLinkage, "arith_sdiv", getModule());
	m_smod = Function::Create(FunctionType::get(Type::Void, arg2Types, false), Linkage::ExternalLinkage, "arith_smod", getModule());
	m_addmod = Function::Create(FunctionType::get(Type::Void, arg3Types, false), Linkage::ExternalLinkage, "arith_addmod", getModule());
	m_mulmod = Function::Create(FunctionType::get(Type::Void, arg3Types, false), Linkage::ExternalLinkage, "arith_mulmod", getModule());
}

Arith256::~Arith256()
{}

llvm::Value* Arith256::binaryOp(llvm::Function* _op, llvm::Value* _arg1, llvm::Value* _arg2)
{
	m_builder.CreateStore(_arg1, m_arg1);
	m_builder.CreateStore(_arg2, m_arg2);
	m_builder.CreateCall3(_op, m_arg1, m_arg2, m_result);
	return m_builder.CreateLoad(m_result);
}

llvm::Value* Arith256::ternaryOp(llvm::Function* _op, llvm::Value* _arg1, llvm::Value* _arg2, llvm::Value* _arg3)
{
	m_builder.CreateStore(_arg1, m_arg1);
	m_builder.CreateStore(_arg2, m_arg2);
	m_builder.CreateStore(_arg3, m_arg3);
	m_builder.CreateCall4(_op, m_arg1, m_arg2, m_arg3, m_result);
	return m_builder.CreateLoad(m_result);
}

llvm::Value* Arith256::mul(llvm::Value* _arg1, llvm::Value* _arg2)
{
	return binaryOp(m_mul, _arg1, _arg2);
}

llvm::Value* Arith256::div(llvm::Value* _arg1, llvm::Value* _arg2)
{
	return binaryOp(m_div, _arg1, _arg2);
}

llvm::Value* Arith256::mod(llvm::Value* _arg1, llvm::Value* _arg2)
{
	return binaryOp(m_mod, _arg1, _arg2);
}

llvm::Value* Arith256::sdiv(llvm::Value* _arg1, llvm::Value* _arg2)
{
	return binaryOp(m_sdiv, _arg1, _arg2);
}

llvm::Value* Arith256::smod(llvm::Value* _arg1, llvm::Value* _arg2)
{
	return binaryOp(m_smod, _arg1, _arg2);
}

llvm::Value* Arith256::addmod(llvm::Value* _arg1, llvm::Value* _arg2, llvm::Value* _arg3)
{
	return ternaryOp(m_addmod, _arg1, _arg2, _arg3);
}

llvm::Value* Arith256::mulmod(llvm::Value* _arg1, llvm::Value* _arg2, llvm::Value* _arg3)
{
	return ternaryOp(m_mulmod, _arg1, _arg2, _arg3);
}


}
}
}


extern "C"
{

	using namespace dev::eth::jit;

	EXPORT void arith_mul(i256* _arg1, i256* _arg2, i256* _result)
	{
		dev::u256 arg1 = llvm2eth(*_arg1);
		dev::u256 arg2 = llvm2eth(*_arg2);
		*_result = eth2llvm(arg1 * arg2);
	}

	EXPORT void arith_div(i256* _arg1, i256* _arg2, i256* _result)
	{
		dev::u256 arg1 = llvm2eth(*_arg1);
		dev::u256 arg2 = llvm2eth(*_arg2);
		*_result = eth2llvm(arg2 == 0 ? arg2 : arg1 / arg2);
	}

	EXPORT void arith_mod(i256* _arg1, i256* _arg2, i256* _result)
	{
		dev::u256 arg1 = llvm2eth(*_arg1);
		dev::u256 arg2 = llvm2eth(*_arg2);
		*_result = eth2llvm(arg2 == 0 ? arg2 : arg1 % arg2);
	}

	EXPORT void arith_sdiv(i256* _arg1, i256* _arg2, i256* _result)
	{
		dev::u256 arg1 = llvm2eth(*_arg1);
		dev::u256 arg2 = llvm2eth(*_arg2);
		*_result = eth2llvm(arg2 == 0 ? arg2 : dev::s2u(dev::u2s(arg1) / dev::u2s(arg2)));
	}

	EXPORT void arith_smod(i256* _arg1, i256* _arg2, i256* _result)
	{
		dev::u256 arg1 = llvm2eth(*_arg1);
		dev::u256 arg2 = llvm2eth(*_arg2);
		*_result = eth2llvm(arg2 == 0 ? arg2 : dev::s2u(dev::u2s(arg1) % dev::u2s(arg2)));
	}

	EXPORT void arith_mulmod(i256* _arg1, i256* _arg2, i256* _arg3, i256* _result)
	{
		dev::u256 arg1 = llvm2eth(*_arg1);
		dev::u256 arg2 = llvm2eth(*_arg2);
		dev::u256 arg3 = llvm2eth(*_arg3);
		*_result = eth2llvm(dev::u256((dev::bigint(arg1) * dev::bigint(arg2)) % arg3));
	}

	EXPORT void arith_addmod(i256* _arg1, i256* _arg2, i256* _arg3, i256* _result)
	{
		dev::u256 arg1 = llvm2eth(*_arg1);
		dev::u256 arg2 = llvm2eth(*_arg2);
		dev::u256 arg3 = llvm2eth(*_arg3);
		*_result = eth2llvm(dev::u256((dev::bigint(arg1) + dev::bigint(arg2)) % arg3));
	}

}


