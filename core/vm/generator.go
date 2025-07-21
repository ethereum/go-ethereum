package vm

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"reflect"
	"runtime"
	"strings"
)

var fSet = token.NewFileSet()

type opFuncNames struct {
	memory    string
	gas       string
	execution string
}

func functionName(f any) string {
	fValue := reflect.ValueOf(f)
	if fValue.IsNil() {
		return ""
	}

	rFunc := runtime.FuncForPC(fValue.Pointer())
	if rFunc == nil {
		panic(fmt.Sprint(f, "is not a function"))
	}
	ident := rFunc.Name()
	arr := strings.Split(ident, ".")
	return arr[len(arr)-1]
}

func collectFuncNames(jt JumpTable) [256]opFuncNames {
	var funcNames [256]opFuncNames
	for opCode, entry := range jt {
		funcNames[opCode] = opFuncNames{
			memory:    functionName(entry.memorySize),
			gas:       functionName(entry.dynamicGas),
			execution: functionName(entry.execute),
		}
	}
	return funcNames
}

type funcDef struct {
	decl   *ast.FuncDecl
	source string
}

type opFuncDefs struct {
	memory    funcDef
	gas       funcDef
	execution funcDef
}

func definitionTargets(name string, funcNames *[256]opFuncNames, funcDefs *[256]opFuncDefs) []*funcDef {
	targets := []*funcDef{}
	for op := range funcNames {
		if funcNames[op].memory == name {
			targets = append(targets, &funcDefs[op].memory)
		}
		if funcNames[op].gas == name {
			targets = append(targets, &funcDefs[op].gas)
		}
		if funcNames[op].execution == name {
			targets = append(targets, &funcDefs[op].execution)
		}
	}
	return targets
}

func extractBody(funcDef string) string {
	start := strings.Index(funcDef, "{")
	end := strings.LastIndex(funcDef, "}")
	return funcDef[start : end+1]
}

func collectFuncDefinitions(funcNames [256]opFuncNames) [256]opFuncDefs {
	pkgs, err := parser.ParseDir(fSet, ".", nil, 0)
	if err != nil {
		panic(fmt.Sprint("failed to parse vm package", err))
	}
	vmPkg := pkgs["vm"]
	if vmPkg == nil {
		panic(fmt.Sprint("vm package not found"))
	}

	var defs [256]opFuncDefs
	for _, astFile := range vmPkg.Files {
		f, err := os.Open(fSet.File(astFile.FileStart).Name())
		if err != nil {
			panic(fmt.Sprint("failed to open file package", astFile.Name.Name, err))
		}

		for _, decl := range astFile.Decls {
			if fDecl, isFunc := decl.(*ast.FuncDecl); isFunc {
				defTargets := definitionTargets(fDecl.Name.Name, &funcNames, &defs)
				if len(defTargets) == 0 {
					continue
				}

				declPos := fSet.Position(fDecl.Pos())
				declEndPos := fSet.Position(fDecl.End())
				declSize := declEndPos.Offset - declPos.Offset

				def := make([]byte, declSize)
				_, err := f.ReadAt(def, int64(declPos.Offset))
				if err != nil && err != io.EOF {
					panic(fmt.Sprint("failed to read function def", fDecl.Name.Name, err))
				}

				for _, defTarget := range defTargets {
					(*defTarget) = funcDef{
						source: string(def),
						decl:   fDecl,
					}
				}
			}
		}
	}

	return defs
}

func outputMemoryCalc(w io.Writer, op *funcDef) {
	fmt.Fprintf(w, `
		_memorySize, _overflow := func (stack *Stack) (uint64, bool) %s(callContext.Stack)
		if _overflow {
			return nil, ErrGasUintOverflow
		}
	`, extractBody(op.source))
}

func outputDynamicGas(w io.Writer, op *funcDef, hasDynamicMemory bool) {
	dynamicMemory := "0"
	if hasDynamicMemory {
		dynamicMemory = "_memorySize"
	}
	fmt.Fprintf(w, `
		_dynamicGas, _gasErr := func (evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (uint64, error) %s(interpreter.evm, callContext.Contract, callContext.Stack, callContext.Memory, %s)
		if _gasErr != nil {
			return nil, ErrOutOfGas
		}
		if !callContext.Contract.UseGas(_dynamicGas, nil, tracing.GasChangeIgnored) {
			return nil, ErrOutOfGas
		}

	`, extractBody(op.source), dynamicMemory)
	if hasDynamicMemory {
		fmt.Fprintf(w, "callContext.Memory.Resize(_memorySize)\n")
	}
}

func outputExecution(w io.Writer, op *funcDef) {
	fmt.Fprintf(w, `
		_, _executionErr := func (pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) %s(&pc, interpreter, callContext)
		if _executionErr != nil {
			return nil, _executionErr
		}
	`, extractBody(op.source))
}

func GenerateRun() {
	set := newLondonInstructionSet()
	funcNames := collectFuncNames(set)
	funcDefinitions := collectFuncDefinitions(funcNames)

	output, _ := os.Create("out.go")

	fmt.Fprintln(output, `package vm
	func Run(interpreter *EVMInterpreter, callContext *ScopeContext) ([]byte, error) {
	
	var pc uint64
	for {
		op := callContext.Contract.GetOp(pc)
	switch op {
	`)
	for opCode := range funcDefinitions {
		fmt.Fprintf(output, `case %d:
			if !callContext.Contract.UseGas(%d, nil, tracing.GasChangeIgnored) {
				return nil, ErrOutOfGas
			}

			if sLen := callContext.Stack.len(); sLen < %d {
				return nil, &ErrStackUnderflow{stackLen: sLen, required: %d}
			} else if sLen > %d {
				return nil, &ErrStackOverflow{stackLen: sLen, limit: %d}
			}
		`, opCode, set[opCode].constantGas, set[opCode].minStack, set[opCode].minStack, set[opCode].maxStack, set[opCode].maxStack)
		if funcDefinitions[opCode].memory.decl != nil {
			outputMemoryCalc(output, &funcDefinitions[opCode].memory)
		} else if len(funcNames[opCode].memory) > 0 {
			panic(fmt.Sprint("failed to get memory handler definition for ", opCodeToString[opCode]))
		}

		if funcDefinitions[opCode].gas.decl != nil {
			outputDynamicGas(output, &funcDefinitions[opCode].gas, funcDefinitions[opCode].memory.decl != nil)
		} else if len(funcNames[opCode].gas) > 0 {
			panic(fmt.Sprint("failed to get gas handler definition for ", opCodeToString[opCode]))
		}

		if funcDefinitions[opCode].execution.decl != nil {
			outputExecution(output, &funcDefinitions[opCode].execution)
		} else if len(funcNames[opCode].execution) > 0 {
			panic(fmt.Sprint("failed to get exec handler definition for ", opCodeToString[opCode]))
		}
	}
	fmt.Fprintln(output, `}
			pc++
	}}`)
	output.Close()
}
