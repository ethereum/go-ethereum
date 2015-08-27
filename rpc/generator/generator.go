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
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
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
	Params   []string
	Return   string
}

func main() {
	// Iterate over all the API files and collect the top level declarations
	api, err := details("github.com/ethereum/go-ethereum/rpc/api")
	if err != nil {
		log.Fatalf("Failed to retrieve API package details: %v", err)
	}
	funs, types, values, err := declarations(api)
	if err != nil {
		log.Fatalf("Failed to collect API declarations: %v", err)
	}
	// Gather all the deteced API endpoints
	methods, err := endpoints(funs, types, values)
	if err != nil {
		log.Fatalf("Failed to gather API endpoints: %v", err)
	}
	// Generate the client API and output if successfull
	if code, err := generate(methods); err != nil {
		log.Fatalf("Failed to format output code: %v", err)
	} else if err := ioutil.WriteFile("../genapi.go", code, 0600); err != nil {
		log.Fatalf("Failed to write output code: %v", err)
	}
}

// details loads the metadata of the Go package.
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

// declarations iterates over a package contents and collects type and value declarations.
func declarations(pack *Package) (map[string]*ast.BlockStmt, map[string]*ast.TypeSpec, map[string]ast.Expr, error) {
	funs := make(map[string]*ast.BlockStmt)
	types := make(map[string]*ast.TypeSpec)
	values := make(map[string]ast.Expr)

	for _, path := range pack.GoFiles {
		// Parse the specified source file
		fileSet := token.NewFileSet()
		tree, err := parser.ParseFile(fileSet, filepath.Join(pack.Dir, path), nil, parser.ParseComments)
		if err != nil {
			return nil, nil, nil, err
		}
		// Collect all top level declarations
		for _, decl := range tree.Decls {
			switch decl := decl.(type) {
			case *ast.FuncDecl:
				if decl.Recv != nil {
					recv := decl.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).String()
					call := decl.Name.String()
					funs[recv+"."+call] = decl.Body
				}
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
	return funs, types, values, nil
}

// returns recursively iterates over a block statement and extracts all the
// return statements that contain nil errors.
func returns(block *ast.BlockStmt) []*ast.ReturnStmt {
	results := []*ast.ReturnStmt{}

	for _, stmt := range block.List {
		switch stmt := stmt.(type) {
		case *ast.ReturnStmt:
			if ident, ok := stmt.Results[1].(*ast.Ident); ok {
				if ident.String() == "nil" {
					results = append(results, stmt)
				}
			}
		case *ast.IfStmt:
			results = append(results, returns(stmt.Body)...)
		}
	}
	return results
}

// endpoints collects the detected RPC API method endpoints.
func endpoints(funs map[string]*ast.BlockStmt, types map[string]*ast.TypeSpec, values map[string]ast.Expr) (map[string][]*Endpoint, error) {
	methods := make(map[string][]*Endpoint)

	// Iterate over all the API mappings, and locate the RPC function associations
	for variable, val := range values {
		if strings.HasSuffix(variable, "Mapping") {
			// Iterate over all the exposed functionality and extract them
			for _, mapping := range val.(*ast.CompositeLit).Elts {
				// Fetch the name of the RPC function, and the associated argument definition
				call := strings.Trim(mapping.(*ast.KeyValueExpr).Key.(*ast.BasicLit).Value, "\"")
				impl := mapping.(*ast.KeyValueExpr).Value.(*ast.SelectorExpr)
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
								return nil, fmt.Errorf("Unknown pointer subtype: %v", sub)
							}
						default:
							return nil, fmt.Errorf("Unknown type: %v", t)
						}
						params = append(params, fmt.Sprintf("%s %s", variable, kind))
						paramList = append(paramList, variable)
					}
				}
				// Try to detect and generate the return type
				owner := impl.X.(*ast.ParenExpr).X.(*ast.StarExpr).X.(*ast.Ident).String()
				method := mapping.(*ast.KeyValueExpr).Value.(*ast.SelectorExpr).Sel.String()
				rets := returns(funs[owner+"."+method])

				result := "interface{}"
				for _, ret := range rets {
					res := ret.Results[0]
					switch res := res.(type) {
					case *ast.Ident:
						if res.String() == "nil" {
							break
						} else if res.String() == "true" || res.String() == "false" {
							result = "bool"
						} else {
							fmt.Println(owner, method, res, "unknown ident")
						}
					case *ast.CallExpr:
						if ident, ok := res.Fun.(*ast.Ident); ok {
							if ident.String() == "string" {
								result = "string"
							} else if ident.String() == "newHexNum" {
								result = "int64"
							} else {
								fmt.Println(owner, method, res, "unknown ident funcion")
							}
						} else {
							fmt.Println(owner, method, res, "call", res.Fun, reflect.TypeOf(res.Fun))
						}
					default:
						fmt.Println(owner, method, res, reflect.TypeOf(res))
					}
				}
				// Insert the function to the submodule collection (alphabetically)
				methods[module] = append(methods[module], &Endpoint{
					Method:   call,
					Function: fmt.Sprintf("%s(%s)", function, strings.Join(params, ",")),
					Params:   paramList,
					Return:   result,
				})
			}
		}
	}
	return methods, nil
}

// generate creates the client code belonging to a set of API method endpoints.
func generate(methods map[string][]*Endpoint) ([]byte, error) {
	// Create a sorted list of API modules and endpoints to generate
	modules := make([]string, 0, len(methods))
	for module, _ := range methods {
		modules = append(modules, module)
	}
	sort.Strings(modules)

	for _, module := range modules {
		for i := 0; i < len(methods[module]); i++ {
			for j := i + 1; j < len(methods[module]); j++ {
				if methods[module][i].Function > methods[module][j].Function {
					methods[module][i], methods[module][j] = methods[module][j], methods[module][i]
				}
			}
		}
	}
	// Start generating the client API
	client := "package rpc\n"

	// Generate the client struct with all its submodules
	client += fmt.Sprintf("type GenApi struct {\n")
	for _, module := range modules {
		client += fmt.Sprintf("%s *%s\n", module, module)
	}
	client += fmt.Sprintf("}\n")

	// Generate the API constructor to create the individual submodules
	client += fmt.Sprint("func NewGenApi(client comms.EthereumClient) *GenApi {\n")
	client += fmt.Sprint("xeth := NewXeth(client)\n\n")
	client += fmt.Sprint("return &GenApi{\n")
	for _, module := range modules {
		client += fmt.Sprintf("%s: &%s{xeth},\n", module, module)
	}
	client += fmt.Sprintf("}}\n")

	// Generate each of the client API calls
	for _, module := range modules {
		endpoints := methods[module]

		client += fmt.Sprintf("\ntype %s struct {\n xeth *Xeth\n}\n", module)
		for _, endpoint := range endpoints {
			// Generate the header (only add variables if conversions are required)
			if endpoint.Return == "interface{}" {
				client += fmt.Sprintf("func (self *%s) %s (interface{}, error) {\n", module, endpoint.Function)
			} else {
				client += fmt.Sprintf("func (self *%s) %s (result %s, failure error) {\n", module, endpoint.Function, endpoint.Return)
			}
			// Generate the actual function invocation (straight return if no return type is known)
			invocation := fmt.Sprintf("self.xeth.Call(\"%s\", nil)", endpoint.Method)
			if len(endpoint.Params) > 0 {
				invocation = fmt.Sprintf("self.xeth.Call(\"%s\", []interface{}{%s})", endpoint.Method, strings.Join(endpoint.Params, ","))
			}
			if endpoint.Return == "interface{}" {
				client += fmt.Sprintf("return %s\n", invocation)
			} else {
				client += fmt.Sprintf("res, err := %s\n", invocation)
			}
			// If conversions are needed, check for errors and post process
			if endpoint.Return != "interface{}" {
				client += fmt.Sprintf("if err != nil { failure = err; return; }\n")

				if endpoint.Return == "int64" {
					client += fmt.Sprintf("return new(big.Int).SetBytes(common.FromHex(res.(string))).Int64(), nil\n")
				} else {
					client += fmt.Sprintf("return res.(%s), nil\n", endpoint.Return)
				}
			}
			client += fmt.Sprintf("}\n")
		}
	}
	// Format the final code and return
	return imports.Process("", []byte(client), nil)
}
