package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"testing"
)

// Package RLP is loaded only once and reused for all tests.
var (
	testFset       = token.NewFileSet()
	testImporter   = importer.ForCompiler(testFset, "source", nil).(types.ImporterFrom)
	testPackageRLP *types.Package
)

func init() {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	testPackageRLP, err = testImporter.ImportFrom(pathOfPackageRLP, cwd, 0)
	if err != nil {
		panic(fmt.Errorf("can't load package RLP: %v", err))
	}
}

var tests = []string{"uints", "nil", "rawvalue", "optional", "bigint"}

func TestOutput(t *testing.T) {
	for _, test := range tests {
		test := test
		t.Run(test, func(t *testing.T) {
			inputFile := filepath.Join("testdata", test+".in.txt")
			outputFile := filepath.Join("testdata", test+".out.txt")
			bctx, typ, err := loadTestSource(inputFile, "Test")
			if err != nil {
				t.Fatal("error loading test source:", err)
			}
			output, err := bctx.generate(typ, true, true)
			if err != nil {
				t.Fatal("error in generate:", err)
			}

			// Set this environment variable to regenerate the test outputs.
			if os.Getenv("WRITE_TEST_FILES") != "" {
				os.WriteFile(outputFile, output, 0644)
			}

			// Check if output matches.
			wantOutput, err := os.ReadFile(outputFile)
			if err != nil {
				t.Fatal("error loading expected test output:", err)
			}
			if !bytes.Equal(output, wantOutput) {
				t.Fatal("output mismatch:\n", string(output))
			}
		})
	}
}

func loadTestSource(file string, typeName string) (*buildContext, *types.Named, error) {
	// Load the test input.
	content, err := os.ReadFile(file)
	if err != nil {
		return nil, nil, err
	}
	f, err := parser.ParseFile(testFset, file, content, 0)
	if err != nil {
		return nil, nil, err
	}
	conf := types.Config{Importer: testImporter}
	pkg, err := conf.Check("test", testFset, []*ast.File{f}, nil)
	if err != nil {
		return nil, nil, err
	}

	// Find the test struct.
	bctx := newBuildContext(testPackageRLP)
	typ, err := lookupStructType(pkg.Scope(), typeName)
	if err != nil {
		return nil, nil, fmt.Errorf("can't find type %s: %v", typeName, err)
	}
	return bctx, typ, nil
}
