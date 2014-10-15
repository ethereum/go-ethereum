
#include <iostream>
#include <fstream>
#include <string>
#include <vector>

#include <boost/algorithm/string.hpp>

#include <llvm/Support/raw_os_ostream.h>

#include <libdevcore/Common.h>
#include <libdevcore/CommonIO.h>
#include <libevmface/Instruction.h>

#include "Compiler.h"
#include "ExecutionEngine.h"


void show_usage()
{
    // FIXME: Use arg[0] as program name?
    std::cerr << "usage: evmcc (-b|-c|-d)+ <inputfile.bc>\n";
}

int main(int argc, char** argv)
{

    std::string input_file;
    bool opt_dissassemble = false;
    bool opt_show_bytes = false;
	bool opt_compile = false;
	bool opt_interpret = false;
    bool opt_unknown = false;

    for (int i = 1; i < argc; i++)
    {
        std::string option = argv[i];
        if (option == "-b")
            opt_show_bytes = true;
        else if (option == "-c")
            opt_compile = true;
        else if (option == "-d")
			opt_dissassemble = true;
		else if (option == "-i")
			opt_interpret = true;
		else if (option[0] != '-' && input_file.empty())
			input_file = option;
        else
        {
            opt_unknown = true;
            break;
        }
    }

    if (opt_unknown ||
        input_file.empty() ||
        (!opt_show_bytes && !opt_compile && !opt_dissassemble && !opt_interpret))
    {
        show_usage();
        exit(1);
    }

    std::ifstream ifs(input_file);
    if (!ifs.is_open())
    {
        std::cerr << "cannot open file " << input_file << std::endl;
        exit(1);
    }

    std::string src((std::istreambuf_iterator<char>(ifs)),
		    (std::istreambuf_iterator<char>()));

    boost::algorithm::trim(src);

	using namespace dev;

    bytes bytecode = fromHex(src);

    if (opt_show_bytes)
    {
        std::cout << dev::memDump(bytecode) << std::endl;
    }

    if (opt_dissassemble)
    {
        std::string assembly = eth::disassemble(bytecode);
        std::cout << assembly << std::endl;
    }

    if (opt_compile)
    {
		auto module = eth::jit::Compiler().compile(bytecode);
		llvm::raw_os_ostream out(std::cout);
		module->print(out, nullptr);
    }

	if (opt_interpret)
	{
		auto engine = eth::jit::ExecutionEngine();
		auto module = eth::jit::Compiler().compile(bytecode);
		module->dump();
		auto result = engine.run(std::move(module));
		return result;
	}

    return 0;
}
