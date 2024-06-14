import { bytesToHex as bufferToHex } from "@nomicfoundation/ethereumjs-util";
import chalk from "chalk";

import {
  CallMessageTrace,
  CreateMessageTrace,
  isCreateTrace,
  isEvmStep,
  isPrecompileTrace,
  MessageTrace,
  PrecompileMessageTrace,
} from "./message-trace";
import { JumpType } from "./model";
import { isJump, isPush, Opcode } from "./opcodes";
import {
  SolidityStackTrace,
  SourceReference,
  StackTraceEntryType,
} from "./solidity-stack-trace";

const MARGIN_SPACE = 6;

export function printMessageTrace(trace: MessageTrace, depth = 0) {
  console.log("");

  if (isCreateTrace(trace)) {
    printCreateTrace(trace, depth);
  } else if (isPrecompileTrace(trace)) {
    printPrecompileTrace(trace, depth);
  } else {
    printCallTrace(trace, depth);
  }

  console.log("");
}

export function printCreateTrace(trace: CreateMessageTrace, depth: number) {
  const margin = "".padStart(depth * MARGIN_SPACE);
  console.log(`${margin}Create trace`);

  if (trace.bytecode !== undefined) {
    console.log(
      `${margin} deploying contract: ${trace.bytecode.contract.location.file.sourceName}:${trace.bytecode.contract.name}`
    );

    console.log(`${margin} code: ${bufferToHex(trace.code)}`);
  } else {
    console.log(
      `${margin} unrecognized deployment code: ${bufferToHex(trace.code)}`
    );
  }

  console.log(`${margin} value: ${trace.value.toString(10)}`);

  if (trace.deployedContract !== undefined) {
    console.log(
      `${margin} contract address: ${bufferToHex(trace.deployedContract)}`
    );
  }

  if (trace.exit.isError()) {
    console.log(`${margin} error: ${trace.exit.getReason()}`);

    // The return data is the deployed-bytecode if there was no error, so we don't show it
    console.log(`${margin} returnData: ${bufferToHex(trace.returnData)}`);
  }

  traceSteps(trace, depth);
}

export function printPrecompileTrace(
  trace: PrecompileMessageTrace,
  depth: number
) {
  const margin = "".padStart(depth * MARGIN_SPACE);
  console.log(`${margin}Precompile trace`);

  console.log(`${margin} precompile number: ${trace.precompile}`);
  console.log(`${margin} value: ${trace.value.toString(10)}`);
  console.log(`${margin} calldata: ${bufferToHex(trace.calldata)}`);

  if (trace.exit.isError()) {
    console.log(`${margin} error: ${trace.exit.getReason()}`);
  }

  console.log(`${margin} returnData: ${bufferToHex(trace.returnData)}`);
}

export function printCallTrace(trace: CallMessageTrace, depth: number) {
  const margin = "".padStart(depth * MARGIN_SPACE);
  console.log(`${margin}Call trace`);

  if (trace.bytecode !== undefined) {
    console.log(
      `${margin} calling contract: ${trace.bytecode.contract.location.file.sourceName}:${trace.bytecode.contract.name}`
    );
  } else {
    console.log(
      `${margin} unrecognized contract code: ${bufferToHex(trace.code)}`
    );
    console.log(`${margin} contract: ${bufferToHex(trace.address)}`);
  }

  console.log(`${margin} value: ${trace.value.toString(10)}`);
  console.log(`${margin} calldata: ${bufferToHex(trace.calldata)}`);

  if (trace.exit.isError()) {
    console.log(`${margin} error: ${trace.exit.getReason()}`);
  }

  console.log(`${margin} returnData: ${bufferToHex(trace.returnData)}`);

  traceSteps(trace, depth);
}

function traceSteps(
  trace: CreateMessageTrace | CallMessageTrace,
  depth: number
) {
  const margin = "".padStart(depth * MARGIN_SPACE);

  console.log(`${margin} steps:`);
  console.log("");

  for (const step of trace.steps) {
    if (isEvmStep(step)) {
      const pc = step.pc.toString(10).padStart(3, "0").padStart(5);

      if (trace.bytecode !== undefined) {
        const inst = trace.bytecode.getInstruction(step.pc);

        let location: string = "";

        if (inst.location !== undefined) {
          location += inst.location.file.sourceName;

          const func = inst.location.getContainingFunction();
          if (func !== undefined) {
            location += `:${
              func.contract?.name ?? func.location.file.sourceName
            }:${func.name}`;
          }

          location += `   -  ${inst.location.offset}:${inst.location.length}`;
        }

        if (isJump(inst.opcode)) {
          const jump =
            inst.jumpType !== JumpType.NOT_JUMP
              ? chalk.bold(`(${JumpType[inst.jumpType]})`)
              : "";

          console.log(
            `${margin}  ${pc}   ${Opcode[inst.opcode]} ${jump}`.padEnd(50),
            location
          );
        } else if (isPush(inst.opcode)) {
          console.log(
            `${margin}  ${pc}   ${Opcode[inst.opcode]} ${bufferToHex(
              inst.pushData!
            )}`.padEnd(50),
            location
          );
        } else {
          console.log(
            `${margin}  ${pc}   ${Opcode[inst.opcode]}`.padEnd(50),
            location
          );
        }
      } else {
        console.log(`${margin}  ${pc}`);
      }
    } else {
      printMessageTrace(step, depth + 1);
    }
  }
}

function flattenSourceReference(sourceReference?: SourceReference) {
  if (sourceReference === undefined) {
    return undefined;
  }

  return {
    ...sourceReference,
    file: sourceReference.sourceName,
  };
}

export function printStackTrace(trace: SolidityStackTrace) {
  const withDecodedMessages = trace.map((entry) =>
    entry.type === StackTraceEntryType.REVERT_ERROR
      ? { ...entry, message: entry.message.decodeError() }
      : entry
  );

  const withHexAddress = withDecodedMessages.map((entry) =>
    "address" in entry
      ? { ...entry, address: bufferToHex(entry.address) }
      : entry
  );

  const withTextualType = withHexAddress.map((entry) => ({
    ...entry,
    type: StackTraceEntryType[entry.type],
  }));

  const withFlattenedSourceReferences = withTextualType.map((entry) => ({
    ...entry,
    sourceReference: flattenSourceReference(entry.sourceReference),
  }));

  console.log(
    JSON.stringify(
      withFlattenedSourceReferences,
      (key, value) => (typeof value === "bigint" ? value.toString() : value),
      2
    )
  );
}
