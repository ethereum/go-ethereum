
#include <chrono>
#include <iostream>
#include <fstream>
#include <ostream>
#include <string>
#include <vector>

#include <boost/algorithm/string.hpp>
#include <boost/program_options.hpp>

#include <llvm/Bitcode/ReaderWriter.h>
#include <llvm/Support/raw_os_ostream.h>
#include <llvm/Support/Signals.h>
#include <llvm/Support/PrettyStackTrace.h>

#include <libdevcore/Common.h>
#include <libdevcore/CommonIO.h>
#include <libevmcore/Instruction.h>
#include <libevm/ExtVMFace.h>
#include <evmjit/libevmjit/Compiler.h>
#include <evmjit/libevmjit/ExecutionEngine.h>


void parseProgramOptions(int _argc, char** _argv, boost::program_options::variables_map& _varMap)
{
	namespace opt = boost::program_options;

	opt::options_description explicitOpts("Allowed options");
	explicitOpts.add_options()
		("help,h", "show usage information")
		("compile,c", "compile the code to LLVM IR")
		("interpret,i", "compile the code to LLVM IR and execute")
		("gas,g", opt::value<size_t>(), "set initial gas for execution")
		("disassemble,d", "dissassemble the code")
		("dump-cfg", "dump control flow graph to graphviz file")
		("dont-optimize", "turn off optimizations")
		("optimize-stack", "optimize stack use between basic blocks (default: on)")
		("rewrite-switch", "rewrite LLVM switch to branches (default: on)")
		("output-ll", opt::value<std::string>(), "dump generated LLVM IR to file")
		("output-bc", opt::value<std::string>(), "dump generated LLVM bitcode to file")
		("show-logs", "output LOG statements to stderr")
		("verbose,V", "enable verbose output");

	opt::options_description implicitOpts("Input files");
	implicitOpts.add_options()
		("input-file", opt::value<std::string>(), "input file");

	opt::options_description allOpts("");
	allOpts.add(explicitOpts).add(implicitOpts);

	opt::positional_options_description inputOpts;
	inputOpts.add("input-file", 1);

	const char* errorMsg = nullptr;
	try
	{
		auto parser = opt::command_line_parser(_argc, _argv).options(allOpts).positional(inputOpts);
		opt::store(parser.run(), _varMap);
		opt::notify(_varMap);
	}
	catch (boost::program_options::error& err)
	{
		errorMsg = err.what();
	}

	if (!errorMsg && _varMap.count("input-file") == 0)
		errorMsg = "missing input file name";

	if (_varMap.count("disassemble") == 0
		&& _varMap.count("compile") == 0
		&& _varMap.count("interpret") == 0)
	{
		errorMsg = "at least one of -c, -i, -d is required";
	}

	if (errorMsg || _varMap.count("help"))
	{
		if (errorMsg)
			std::cerr << "Error: " << errorMsg << std::endl;

		std::cout << "Usage: " << _argv[0] << " <options> input-file " << std::endl
				  << explicitOpts << std::endl;
		std::exit(errorMsg ? 1 : 0);
	}
}

int main(int argc, char** argv)
{
	llvm::sys::PrintStackTraceOnErrorSignal();
	llvm::PrettyStackTraceProgram X(argc, argv);

	boost::program_options::variables_map options;
	parseProgramOptions(argc, argv, options);

	auto inputFile = options["input-file"].as<std::string>();
	std::ifstream ifs(inputFile);
	if (!ifs.is_open())
	{
		std::cerr << "cannot open input file " << inputFile << std::endl;
		exit(1);
	}

	std::string src((std::istreambuf_iterator<char>(ifs)),
					(std::istreambuf_iterator<char>()));

	boost::algorithm::trim(src);

	using namespace dev;

	bytes bytecode = fromHex(src);

	if (options.count("disassemble"))
	{
		std::string assembly = eth::disassemble(bytecode);
		std::cout << assembly << std::endl;
	}

	if (options.count("compile") || options.count("interpret"))
	{
		size_t initialGas = 10000;

		if (options.count("gas"))
			initialGas = options["gas"].as<size_t>();

		auto compilationStartTime = std::chrono::high_resolution_clock::now();

		eth::jit::Compiler::Options compilerOptions;
		compilerOptions.dumpCFG = options.count("dump-cfg") > 0;
		bool optimize = options.count("dont-optimize") == 0;
		compilerOptions.optimizeStack = optimize || options.count("optimize-stack") > 0;
		compilerOptions.rewriteSwitchToBranches = optimize || options.count("rewrite-switch") > 0;

		auto compiler = eth::jit::Compiler(compilerOptions);
		auto module = compiler.compile(bytecode, "main");

		auto compilationEndTime = std::chrono::high_resolution_clock::now();

		module->dump();

		if (options.count("output-ll"))
		{
			auto outputFile = options["output-ll"].as<std::string>();
			std::ofstream ofs(outputFile);
			if (!ofs.is_open())
			{
				std::cerr << "cannot open output file " << outputFile << std::endl;
				exit(1);
			}
			llvm::raw_os_ostream ros(ofs);
			module->print(ros, nullptr);
			ofs.close();
		}

		if (options.count("output-bc"))
		{
			auto outputFile = options["output-bc"].as<std::string>();
			std::ofstream ofs(outputFile);
			if (!ofs.is_open())
			{
				std::cerr << "cannot open output file " << outputFile << std::endl;
				exit(1);
			}
			llvm::raw_os_ostream ros(ofs);
			llvm::WriteBitcodeToFile(module.get(), ros);
			ros.flush();
			ofs.close();
		}

		if (options.count("verbose"))
		{
			std::cerr << "*** Compilation time: "
					  << std::chrono::duration_cast<std::chrono::microseconds>(compilationEndTime - compilationStartTime).count()
					  << std::endl;
		}

		if (options.count("interpret"))
		{
			using namespace eth::jit;

			ExecutionEngine engine;
			eth::jit::u256 gas = initialGas;

			// Create random runtime data
			RuntimeData data;
			data.set(RuntimeData::Gas, gas);
			data.set(RuntimeData::Address, (u160)Address(1122334455667788));
			data.set(RuntimeData::Caller, (u160)Address(0xfacefacefaceface));
			data.set(RuntimeData::Origin, (u160)Address(101010101010101010));
			data.set(RuntimeData::CallValue, 0xabcd);
			data.set(RuntimeData::CallDataSize, 3);
			data.set(RuntimeData::GasPrice, 1003);
			data.set(RuntimeData::CoinBase, (u160)Address(101010101010101015));
			data.set(RuntimeData::TimeStamp, 1005);
			data.set(RuntimeData::Number, 1006);
			data.set(RuntimeData::Difficulty, 16);
			data.set(RuntimeData::GasLimit, 1008);
			data.set(RuntimeData::CodeSize, bytecode.size());
			data.callData = (uint8_t*)"abc";
			data.code = bytecode.data();

			// BROKEN: env_* functions must be implemented & RuntimeData struct created
			// TODO: Do not compile module again
			auto result = engine.run(bytecode, &data, nullptr);
			return static_cast<int>(result);
		}
	}

	return 0;
}
