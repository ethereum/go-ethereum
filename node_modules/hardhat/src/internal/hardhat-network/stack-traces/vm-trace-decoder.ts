import chalk from "chalk";
import debug from "debug";
import { Reporter } from "../../sentry/reporter";
import { TracingConfig } from "../provider/node-types";
import { createModelsAndDecodeBytecodes } from "./compiler-to-model";
import { ContractsIdentifier } from "./contracts-identifier";
import {
  isCreateTrace,
  isEvmStep,
  isPrecompileTrace,
  MessageTrace,
} from "./message-trace";
import { Bytecode, ContractFunctionType } from "./model";
import {
  FALLBACK_FUNCTION_NAME,
  RECEIVE_FUNCTION_NAME,
  UNRECOGNIZED_CONTRACT_NAME,
  UNRECOGNIZED_FUNCTION_NAME,
} from "./solidity-stack-trace";

const log = debug("hardhat:core:hardhat-network:node");

export class VmTraceDecoder {
  constructor(private readonly _contractsIdentifier: ContractsIdentifier) {}

  public getContractAndFunctionNamesForCall(
    code: Buffer,
    calldata?: Buffer
  ): { contractName: string; functionName?: string } {
    const isCreate = calldata === undefined;
    const bytecode = this._contractsIdentifier.getBytecodeForCall(
      code,
      isCreate
    );

    const contractName = bytecode?.contract.name ?? UNRECOGNIZED_CONTRACT_NAME;

    if (isCreate) {
      return {
        contractName,
      };
    } else {
      if (bytecode === undefined) {
        return {
          contractName,
          functionName: "",
        };
      } else {
        const func = bytecode.contract.getFunctionFromSelector(
          calldata.slice(0, 4)
        );

        const functionName: string =
          func === undefined
            ? UNRECOGNIZED_FUNCTION_NAME
            : func.type === ContractFunctionType.FALLBACK
            ? FALLBACK_FUNCTION_NAME
            : func.type === ContractFunctionType.RECEIVE
            ? RECEIVE_FUNCTION_NAME
            : func.name;

        return {
          contractName,
          functionName,
        };
      }
    }
  }

  public tryToDecodeMessageTrace(messageTrace: MessageTrace): MessageTrace {
    if (isPrecompileTrace(messageTrace)) {
      return messageTrace;
    }

    return {
      ...messageTrace,
      bytecode: this._contractsIdentifier.getBytecodeForCall(
        messageTrace.code,
        isCreateTrace(messageTrace)
      ),
      steps: messageTrace.steps.map((s) =>
        isEvmStep(s) ? s : this.tryToDecodeMessageTrace(s)
      ),
    };
  }

  public addBytecode(bytecode: Bytecode) {
    this._contractsIdentifier.addBytecode(bytecode);
  }
}

export function initializeVmTraceDecoder(
  vmTraceDecoder: VmTraceDecoder,
  tracingConfig: TracingConfig
) {
  if (tracingConfig.buildInfos === undefined) {
    return;
  }

  try {
    for (const buildInfo of tracingConfig.buildInfos) {
      const bytecodes = createModelsAndDecodeBytecodes(
        buildInfo.solcVersion,
        buildInfo.input,
        buildInfo.output
      );

      for (const bytecode of bytecodes) {
        if (
          tracingConfig.ignoreContracts === true &&
          bytecode.contract.name.startsWith("Ignored")
        ) {
          continue;
        }

        vmTraceDecoder.addBytecode(bytecode);
      }
    }
  } catch (error) {
    console.warn(
      chalk.yellow(
        "The Hardhat Network tracing engine could not be initialized. Run Hardhat with --verbose to learn more."
      )
    );

    log(
      "Hardhat Network tracing disabled: ContractsIdentifier failed to be initialized. Please report this to help us improve Hardhat.\n",
      error
    );

    if (error instanceof Error) {
      Reporter.reportError(error);
    }
  }
}
