
#include <chrono>
#include <iostream>
#include <fstream>
#include <ostream>
#include <string>
#include <vector>

#include <boost/algorithm/string.hpp>

#include <llvm/Support/raw_os_ostream.h>

#include <libdevcore/Common.h>
#include <libdevcore/CommonIO.h>
#include <libevmface/Instruction.h>
#include <libevmjit/Compiler.h>
#include <libevmjit/ExecutionEngine.h>


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
	bool opt_dump_graph = false;
	bool opt_unknown = false;
	bool opt_verbose = false;
	size_t initialGas = 10000;

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
		else if (option == "--dump-cfg")
			opt_dump_graph = true;
		else if (option == "-g" && i + 1 < argc)
		{
			std::string gasValue = argv[++i];
			initialGas = boost::lexical_cast<size_t>(gasValue);
			std::cerr << "Initial gas set to " << initialGas << "\n";
		}
		else if (option == "-v")
			opt_verbose = true;
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
		std::cout << memDump(bytecode) << std::endl;

	if (opt_dissassemble)
	{
		std::string assembly = eth::disassemble(bytecode);
		std::cout << assembly << std::endl;
	}

	if (opt_compile || opt_interpret)
	{
		auto compilationStartTime = std::chrono::high_resolution_clock::now();

		auto compiler = eth::jit::Compiler();
		auto module = compiler.compile({bytecode.data(), bytecode.size()});

		auto compilationEndTime = std::chrono::high_resolution_clock::now();

		module->dump();

		if (opt_verbose)
		{
			std::cerr << "*** Compilation time: "
			          << std::chrono::duration_cast<std::chrono::microseconds>(compilationEndTime - compilationStartTime).count()
			          << std::endl;
		}

		if (opt_dump_graph)
		{
			std::ofstream ofs("blocks.dot");
			compiler.dumpBasicBlockGraph(ofs);
			ofs.close();
			std::cout << "Basic blocks graph written to block.dot\n";
		}

		if (opt_interpret)
		{
			auto engine = eth::jit::ExecutionEngine();
			u256 gas = initialGas;
			auto result = engine.run(std::move(module), gas);
			return result;
		}
	}

	return 0;
}
