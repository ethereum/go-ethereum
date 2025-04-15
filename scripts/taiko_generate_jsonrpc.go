package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/eth"
)

type JRGenSpec struct {
	Schema      string                 `json:"$schema"`
	JRGen       string                 `json:"jrgen"`
	JSONRPC     string                 `json:"jsonrpc"`
	Info        Info                   `json:"info"`
	Definitions map[string]interface{} `json:"definitions"`
	Methods     map[string]Method      `json:"methods"`
}

type Info struct {
	Title       string   `json:"title"`
	Description []string `json:"description"`
	Version     string   `json:"version"`
	Servers     []Server `json:"servers,omitempty"`
}

type Server struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

type Method struct {
	Summary     string                 `json:"summary"`
	Description string                 `json:"description"`
	Tags        []string               `json:"tags,omitempty"`
	Params      map[string]interface{} `json:"params,omitempty"`
	Result      map[string]interface{} `json:"result,omitempty"`
}

func parseParamNamesFromFile(filename string) (map[string][]string, map[string]string, error) {
	fset := token.NewFileSet()
	fileAST, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, err
	}

	methodParams := make(map[string][]string)
	methodDocs := make(map[string]string)

	for _, decl := range fileAST.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok && funcDecl.Recv != nil {
			recvField := funcDecl.Recv.List[0]
			var recvTypeStr string

			switch t := recvField.Type.(type) {
			case *ast.StarExpr:
				if ident, ok := t.X.(*ast.Ident); ok {
					recvTypeStr = "*" + ident.Name
				}
			case *ast.Ident:
				recvTypeStr = t.Name
			}

			if recvTypeStr == "*TaikoAPIBackend" || recvTypeStr == "*TaikoAuthAPIBackend" {
				var params []string
				if funcDecl.Type.Params != nil {
					for _, field := range funcDecl.Type.Params.List {
						if len(field.Names) > 0 {
							for _, name := range field.Names {
								params = append(params, name.Name)
							}
						} else {
							params = append(params, exprToString(field.Type))
						}
					}
				}
				key := recvTypeStr + "." + funcDecl.Name.Name
				methodParams[key] = params

				if funcDecl.Doc != nil {
					methodDocs[key] = strings.TrimSpace(funcDecl.Doc.Text())
				}
			}
		}
	}

	return methodParams, methodDocs, nil
}

func exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprToString(t.X)
	case *ast.ArrayType:
		return "[]" + exprToString(t.Elt)
	case *ast.SelectorExpr:
		return exprToString(t.X) + "." + t.Sel.Name
	default:
		return fmt.Sprintf("%T", expr)
	}
}

func generateReturnSchema(goType reflect.Type) map[string]interface{} {
	switch goType.Kind() {
	case reflect.Ptr:
		return map[string]interface{}{
			"$ref": "#/definitions/" + goType.String(),
		}
	case reflect.Slice:
		elem := goType.Elem()
		if elem.Kind() == reflect.Ptr {
			return map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"$ref": "#/definitions/" + elem.String(),
				},
			}
		}
		return map[string]interface{}{
			"type":  "array",
			"items": generateTypeRef(elem),
		}
	default:
		return generateTypeRef(goType)
	}
}

func generateTypeRef(goType reflect.Type) map[string]interface{} {
	switch goType.Kind() {
	case reflect.Ptr:
		switch goType.String() {
		case "*miner.PreBuiltTxList":
			return map[string]interface{}{
				"$ref": "#/definitions/*miner.PreBuiltTxList",
			}
		case "*big.Int":
			return map[string]interface{}{
				"$ref": "#/definitions/*big.Int",
			}
		case "*rawdb.L1Origin":
			return map[string]interface{}{
				"$ref": "#/definitions/*rawdb.L1Origin",
			}
		case "*math.HexOrDecimal256":
			return map[string]interface{}{
				"$ref": "#/definitions/*math.HexOrDecimal256",
			}
		}
	case reflect.Slice:
		elem := goType.Elem()
		if elem.Kind() == reflect.Ptr {
			return map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"$ref": "#/definitions/" + elem.String(),
				},
			}
		}
		return map[string]interface{}{
			"type":  "array",
			"items": generateTypeRef(elem),
		}
	case reflect.String:
		return map[string]interface{}{
			"type":     "string",
			"examples": []string{"0x123456"},
		}
	case reflect.Int, reflect.Int64, reflect.Uint, reflect.Uint64:
		return map[string]interface{}{
			"type":     "integer",
			"examples": []int{10000},
		}
	case reflect.Bool:
		return map[string]interface{}{
			"type":     "boolean",
			"examples": []bool{true},
		}
	}

	return map[string]interface{}{
		"type":     "string",
		"examples": []string{"0x123456"},
	}
}

func extractMethods(obj interface{}, prefix string, paramMapping map[string][]string, docMapping map[string]string) map[string]Method {
	t := reflect.TypeOf(obj)
	methods := make(map[string]Method)

	var recName string
	if t.Kind() == reflect.Ptr {
		recName = "*" + t.Elem().Name()
	} else {
		recName = t.Name()
	}

	for i := 0; i < t.NumMethod(); i++ {
		m := t.Method(i)
		key := recName + "." + m.Name
		mappedNames := paramMapping[key]

		props := map[string]interface{}{}
		required := []string{}

		for j := 1; j < m.Type.NumIn(); j++ {
			paramType := m.Type.In(j)
			paramName := fmt.Sprintf("param%d", j-1)
			if j-1 < len(mappedNames) {
				paramName = mappedNames[j-1]
			}
			props[paramName] = generateTypeRef(paramType)
			required = append(required, paramName)
		}

		var paramsObj map[string]interface{}
		if len(props) > 0 {
			paramsObj = map[string]interface{}{
				"type":       "object",
				"properties": props,
				"required":   required,
			}
		}

		var resultType map[string]interface{}
		if m.Type.NumOut() > 0 {
			out := m.Type.Out(0)
			resultType = generateReturnSchema(out)
		}

		description := fmt.Sprintf("Invokes the %s method on %s", m.Name, recName)
		if doc, ok := docMapping[key]; ok {
			description = doc
		}

		method := Method{
			Summary:     fmt.Sprintf("RPC method %s", m.Name),
			Description: description,
			Tags:        []string{prefix},
			Params:      paramsObj,
			Result:      resultType,
		}

		methods[prefix+m.Name] = method
	}
	return methods
}

func main() {
	paramMapping, docMapping, err := parseParamNamesFromFile("eth/taiko_api_backend.go")
	if err != nil {
		log.Fatalf("Failed to parse params: %v", err)
	}

	spec := JRGenSpec{
		Schema:  "https://rawgit.com/mzernetsch/jrgen/master/jrgen-spec.schema.json",
		JRGen:   "1.2",
		JSONRPC: "2.0",
		Info: Info{
			Title:       "Taiko JSON-RPC API",
			Version:     "1.0",
			Description: []string{"Auto-generated JSON-RPC API for Taiko backend."},
		},
		Definitions: map[string]interface{}{
			"*miner.PreBuiltTxList": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"TxList":           map[string]interface{}{"type": "array"},
					"EstimatedGasUsed": map[string]interface{}{"type": "integer", "examples": []int{10000}},
					"BytesLength":      map[string]interface{}{"type": "integer", "examples": []int{10000}},
				},
				"required": []string{"TxList", "EstimatedGasUsed", "BytesLength"},
			},
			"*big.Int": map[string]interface{}{
				"type":     "integer",
				"examples": []int{10000},
			},
			"*rawdb.L1Origin": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"blockID":       map[string]interface{}{"$ref": "#/definitions/*big.Int"},
					"l2BlockHash":   map[string]interface{}{"type": "string", "examples": []string{"0x123456"}},
					"l1BlockHeight": map[string]interface{}{"$ref": "#/definitions/*big.Int"},
					"l1BlockHash":   map[string]interface{}{"type": "string", "examples": []string{"0x123456"}},
				},
				"required": []string{"blockID", "l2BlockHash", "l1BlockHeight", "l1BlockHash"},
			},
			"*math.HexOrDecimal256": map[string]interface{}{
				"$ref":        "#/definitions/*big.Int",
				"description": "Hexadecimal or decimal representation of a number.",
			},
		},
		Methods: map[string]Method{},
	}

	taiko := &eth.TaikoAPIBackend{}
	auth := &eth.TaikoAuthAPIBackend{}

	for k, v := range extractMethods(taiko, "taiko_", paramMapping, docMapping) {
		spec.Methods[k] = v
	}
	for k, v := range extractMethods(auth, "taikoAuth_", paramMapping, docMapping) {
		spec.Methods[k] = v
	}

	jsonData, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}
	err = os.WriteFile("jrgen.json", jsonData, 0644)
	if err != nil {
		log.Fatalf("Failed to write file: %v", err)
	}

	fmt.Println("jrgen.json generated successfully.")
}
