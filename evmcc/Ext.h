
#include <llvm/IR/IRBuilder.h>

#include <libevm/ExtVMFace.h>

namespace evmcc
{



class Ext
{
public:
	Ext(llvm::IRBuilder<>& _builder);
	static void init(std::unique_ptr<dev::eth::ExtVMFace> _ext);

private:
	llvm::IRBuilder<>& m_builder;
};
	

}