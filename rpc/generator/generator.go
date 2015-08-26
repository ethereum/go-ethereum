// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// This is the Go API auto-generator for the RPC APIs.
package main //build !none

//go:generate go run generator.go

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"

	"golang.org/x/tools/imports"
)

// Package description from the go list command
type Package struct {
	Name    string
	Dir     string
	GoFiles []string
}

type Endpoint struct {
	Method   string
	Function string
	Argument string
	Params   []string
}

func main() {
	// Iterate over all the API files and collect the top level declarations
	api, err := details("github.com/ethereum/go-ethereum/rpc/api")
	if err != nil {
		log.Fatalf("Failed to retrieve API package details: %v", err)
	}
	types, values, err := declarations(api)
	if err != nil {
		log.Fatalf("Failed to collect API declarations: %v", err)
	}
	// Create the collectors for the sobmodules
	modules := make(map[string][]*Endpoint)

	// Iterate over all the API mappings, and locate the RPC function associations
	for variable, val := range values {
		if strings.HasSuffix(variable, "Mapping") {
			for _, mapping := range val.(*ast.CompositeLit).Elts {
				// Fetch the name of the RPC function, and the associated argument definition
				call := strings.Trim(mapping.(*ast.KeyValueExpr).Key.(*ast.BasicLit).Value, "\"")
				args := mapping.(*ast.KeyValueExpr).Value.(*ast.SelectorExpr).Sel.Name + "Args"

				// Generate the submodule and function names
				module := strings.Split(call, "_")[0]
				module = string(unicode.ToUpper(rune(module[0]))) + module[1:]
				function := strings.Split(call, "_")[1]
				function = string(unicode.ToUpper(rune(function[0]))) + function[1:]

				// Generate the parameter list
				params, paramList := []string{}, []string{}
				if arg := types[args]; arg != nil {
					for _, field := range arg.Type.(*ast.StructType).Fields.List {
						variable := field.Names[0].String()
						variable = string(unicode.ToLower(rune(variable[0]))) + variable[1:]

						kind := ""
						switch t := field.Type.(type) {
						case *ast.Ident:
							kind = t.String()
						case *ast.SelectorExpr:
							kind = t.X.(*ast.Ident).String() + "." + t.Sel.String()
						case *ast.StarExpr:
							switch sub := t.X.(type) {
							case *ast.Ident:
								kind = "*" + sub.String()
							case *ast.SelectorExpr:
								kind = "*" + sub.X.(*ast.Ident).String() + "." + sub.Sel.String()
							default:
								log.Fatalf("Unknown pointer subtype: %v", sub)
							}
						default:
							log.Fatalf("Unknown type: %v", t)
						}
						params = append(params, fmt.Sprintf("%s %s", variable, kind))
						paramList = append(paramList, variable)
					}
				}
				// Append the function to the sobmodule collection
				modules[module] = append(modules[module], &Endpoint{
					Method:   call,
					Function: fmt.Sprintf("%s(%s)", function, strings.Join(params, ",")),
					Argument: args,
					Params:   paramList,
				})
			}
		}
	}
	// Start generating the client API
	client := "package rpc\n"

	// Generate the client struct with all its submodules
	client += fmt.Sprintf("type GenApi struct {\n")
	for module, _ := range modules {
		client += fmt.Sprintf("%s *%s\n", module, module)
	}
	client += fmt.Sprintf("}\n")

	// Generate the API constructor to create the individual submodules
	client += fmt.Sprint("func NewGenApi(xeth *Xeth) *GenApi {\n")
	client += fmt.Sprint("return &GenApi{\n")
	for module, _ := range modules {
		client += fmt.Sprintf("%s: &%s{xeth},\n", module, module)
	}
	client += fmt.Sprintf("}}\n")

	// Generate each of the client API calls
	for module, endpoints := range modules {
		client += fmt.Sprintf("\ntype %s struct {\n xeth *Xeth\n}\n", module)
		for _, endpoint := range endpoints {
			client += fmt.Sprintf("func (self *%s) %s (interface{}, error) {\n", module, endpoint.Function)
			if len(endpoint.Params) == 0 {
				client += fmt.Sprintf("return self.xeth.Call(\"%s\", nil)\n", endpoint.Method)
			} else {
				client += fmt.Sprintf("return self.xeth.Call(\"%s\", []interface{}{%s})\n", endpoint.Method, strings.Join(endpoint.Params, ","))
			}
			client += fmt.Sprintf("}\n")
		}
	}
	// Format the final code and output
	if blob, err := imports.Process("", []byte(client), nil); err != nil {
		log.Fatalf("Failed to format output code: %v", err)
	} else {
		fmt.Println(string(blob))
	}
}

// Loads the details of the Go package.
func details(name string) (*Package, error) {
	// Create the command to retrieve the package infos
	cmd := exec.Command("go", "list", "-e", "-json", name)

	// Retrieve the output, redirect the errors
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = os.Stderr

	// Start executing and parse the results
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	defer cmd.Process.Kill()

	info := new(Package)
	if err := json.NewDecoder(out).Decode(&info); err != nil {
		return nil, err
	}
	// Clean up and return
	if err := cmd.Wait(); err != nil {
		return nil, err
	}
	return info, nil
}

// Iterates over a package contents and collects interesting declarations.
func declarations(pack *Package) (map[string]*ast.TypeSpec, map[string]ast.Expr, error) {
	types := make(map[string]*ast.TypeSpec)
	values := make(map[string]ast.Expr)

	for _, path := range pack.GoFiles {
		// Parse the specified source file
		fileSet := token.NewFileSet()
		tree, err := parser.ParseFile(fileSet, filepath.Join(pack.Dir, path), nil, parser.ParseComments)
		if err != nil {
			return nil, nil, err
		}
		// Collect all top level declarations
		for _, decl := range tree.Decls {
			switch decl := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range decl.Specs {
					switch spec := spec.(type) {
					case *ast.ValueSpec:
						for i, name := range spec.Names {
							values[name.String()] = spec.Values[i]
						}
					case *ast.TypeSpec:
						types[spec.Name.String()] = spec
					}
				}
			}
		}
	}
	return types, values, nil
}
