package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/robertkrimen/otto/ast"
)

func marshal(name string, children ...interface{}) interface{} {
	if len(children) == 1 {
		if name == "" {
			return testMarshalNode(children[0])
		}
		return map[string]interface{}{
			name: children[0],
		}
	}
	map_ := map[string]interface{}{}
	length := len(children) / 2
	for i := 0; i < length; i++ {
		name := children[i*2].(string)
		value := children[i*2+1]
		map_[name] = value
	}
	if name == "" {
		return map_
	}
	return map[string]interface{}{
		name: map_,
	}
}

func testMarshalNode(node interface{}) interface{} {
	switch node := node.(type) {

	// Expression

	case *ast.ArrayLiteral:
		return marshal("Array", testMarshalNode(node.Value))

	case *ast.AssignExpression:
		return marshal("Assign",
			"Left", testMarshalNode(node.Left),
			"Right", testMarshalNode(node.Right),
		)

	case *ast.BinaryExpression:
		return marshal("BinaryExpression",
			"Operator", node.Operator.String(),
			"Left", testMarshalNode(node.Left),
			"Right", testMarshalNode(node.Right),
		)

	case *ast.BooleanLiteral:
		return marshal("Literal", node.Value)

	case *ast.CallExpression:
		return marshal("Call",
			"Callee", testMarshalNode(node.Callee),
			"ArgumentList", testMarshalNode(node.ArgumentList),
		)

	case *ast.ConditionalExpression:
		return marshal("Conditional",
			"Test", testMarshalNode(node.Test),
			"Consequent", testMarshalNode(node.Consequent),
			"Alternate", testMarshalNode(node.Alternate),
		)

	case *ast.DotExpression:
		return marshal("Dot",
			"Left", testMarshalNode(node.Left),
			"Member", node.Identifier.Name,
		)

	case *ast.NewExpression:
		return marshal("New",
			"Callee", testMarshalNode(node.Callee),
			"ArgumentList", testMarshalNode(node.ArgumentList),
		)

	case *ast.NullLiteral:
		return marshal("Literal", nil)

	case *ast.NumberLiteral:
		return marshal("Literal", node.Value)

	case *ast.ObjectLiteral:
		return marshal("Object", testMarshalNode(node.Value))

	case *ast.RegExpLiteral:
		return marshal("Literal", node.Literal)

	case *ast.StringLiteral:
		return marshal("Literal", node.Literal)

	case *ast.VariableExpression:
		return []interface{}{node.Name, testMarshalNode(node.Initializer)}

	// Statement

	case *ast.Program:
		return testMarshalNode(node.Body)

	case *ast.BlockStatement:
		return marshal("BlockStatement", testMarshalNode(node.List))

	case *ast.EmptyStatement:
		return "EmptyStatement"

	case *ast.ExpressionStatement:
		return testMarshalNode(node.Expression)

	case *ast.ForInStatement:
		return marshal("ForIn",
			"Into", marshal("", node.Into),
			"Source", marshal("", node.Source),
			"Body", marshal("", node.Body),
		)

	case *ast.FunctionLiteral:
		return marshal("Function", testMarshalNode(node.Body))

	case *ast.Identifier:
		return marshal("Identifier", node.Name)

	case *ast.IfStatement:
		if_ := marshal("",
			"Test", testMarshalNode(node.Test),
			"Consequent", testMarshalNode(node.Consequent),
		).(map[string]interface{})
		if node.Alternate != nil {
			if_["Alternate"] = testMarshalNode(node.Alternate)
		}
		return marshal("If", if_)

	case *ast.LabelledStatement:
		return marshal("Label",
			"Name", node.Label.Name,
			"Statement", testMarshalNode(node.Statement),
		)
	case ast.Property:
		return marshal("",
			"Key", node.Key,
			"Value", testMarshalNode(node.Value),
		)

	case *ast.ReturnStatement:
		return marshal("Return", testMarshalNode(node.Argument))

	case *ast.SequenceExpression:
		return marshal("Sequence", testMarshalNode(node.Sequence))

	case *ast.ThrowStatement:
		return marshal("Throw", testMarshalNode(node.Argument))

	case *ast.VariableStatement:
		return marshal("Var", testMarshalNode(node.List))

	}

	{
		value := reflect.ValueOf(node)
		if value.Kind() == reflect.Slice {
			tmp0 := []interface{}{}
			for index := 0; index < value.Len(); index++ {
				tmp0 = append(tmp0, testMarshalNode(value.Index(index).Interface()))
			}
			return tmp0
		}
	}

	if node != nil {
		fmt.Fprintf(os.Stderr, "testMarshalNode(%T)\n", node)
	}

	return nil
}

func testMarshal(node interface{}) string {
	value, err := json.Marshal(testMarshalNode(node))
	if err != nil {
		panic(err)
	}
	return string(value)
}

func TestParserAST(t *testing.T) {
	tt(t, func() {

		test := func(inputOutput string) {
			match := matchBeforeAfterSeparator.FindStringIndex(inputOutput)
			input := strings.TrimSpace(inputOutput[0:match[0]])
			wantOutput := strings.TrimSpace(inputOutput[match[1]:])
			_, program, err := testParse(input)
			is(err, nil)
			haveOutput := testMarshal(program)
			tmp0, tmp1 := bytes.Buffer{}, bytes.Buffer{}
			json.Indent(&tmp0, []byte(haveOutput), "\t\t", "   ")
			json.Indent(&tmp1, []byte(wantOutput), "\t\t", "   ")
			is("\n\t\t"+tmp0.String(), "\n\t\t"+tmp1.String())
		}

		test(`
        ---
[]
        `)

		test(`
        ;
        ---
[
  "EmptyStatement"
]
        `)

		test(`
        ;;;
        ---
[
  "EmptyStatement",
  "EmptyStatement",
  "EmptyStatement"
]
        `)

		test(`
        1; true; abc; "abc"; null;
        ---
[
  {
    "Literal": 1
  },
  {
    "Literal": true
  },
  {
    "Identifier": "abc"
  },
  {
    "Literal": "\"abc\""
  },
  {
    "Literal": null
  }
]
        `)

		test(`
        { 1; null; 3.14159; ; }
        ---
[
  {
    "BlockStatement": [
      {
        "Literal": 1
      },
      {
        "Literal": null
      },
      {
        "Literal": 3.14159
      },
      "EmptyStatement"
    ]
  }
]
        `)

		test(`
        new abc();
        ---
[
  {
    "New": {
      "ArgumentList": [],
      "Callee": {
        "Identifier": "abc"
      }
    }
  }
]
        `)

		test(`
        new abc(1, 3.14159)
        ---
[
  {
    "New": {
      "ArgumentList": [
        {
          "Literal": 1
        },
        {
          "Literal": 3.14159
        }
      ],
      "Callee": {
        "Identifier": "abc"
      }
    }
  }
]
        `)

		test(`
        true ? false : true
        ---
[
  {
    "Conditional": {
      "Alternate": {
        "Literal": true
      },
      "Consequent": {
        "Literal": false
      },
      "Test": {
        "Literal": true
      }
    }
  }
]
        `)

		test(`
        true || false
        ---
[
  {
    "BinaryExpression": {
      "Left": {
        "Literal": true
      },
      "Operator": "||",
      "Right": {
        "Literal": false
      }
    }
  }
]
        `)

		test(`
        0 + { abc: true }
        ---
[
  {
    "BinaryExpression": {
      "Left": {
        "Literal": 0
      },
      "Operator": "+",
      "Right": {
        "Object": [
          {
            "Key": "abc",
            "Value": {
              "Literal": true
            }
          }
        ]
      }
    }
  }
]
        `)

		test(`
        1 == "1"
        ---
[
  {
    "BinaryExpression": {
      "Left": {
        "Literal": 1
      },
      "Operator": "==",
      "Right": {
        "Literal": "\"1\""
      }
    }
  }
]
        `)

		test(`
        abc(1)
        ---
[
  {
    "Call": {
      "ArgumentList": [
        {
          "Literal": 1
        }
      ],
      "Callee": {
        "Identifier": "abc"
      }
    }
  }
]
        `)

		test(`
        Math.pow(3, 2)
        ---
[
  {
    "Call": {
      "ArgumentList": [
        {
          "Literal": 3
        },
        {
          "Literal": 2
        }
      ],
      "Callee": {
        "Dot": {
          "Left": {
            "Identifier": "Math"
          },
          "Member": "pow"
        }
      }
    }
  }
]
        `)

		test(`
        1, 2, 3
        ---
[
  {
    "Sequence": [
      {
        "Literal": 1
      },
      {
        "Literal": 2
      },
      {
        "Literal": 3
      }
    ]
  }
]
        `)

		test(`
        / abc /   gim;
        ---
[
  {
    "Literal": "/ abc /   gim"
  }
]
        `)

		test(`
        if (0)
            1;
        ---
[
  {
    "If": {
      "Consequent": {
        "Literal": 1
      },
      "Test": {
        "Literal": 0
      }
    }
  }
]
        `)

		test(`
        0+function(){
            return;
        }
        ---
[
  {
    "BinaryExpression": {
      "Left": {
        "Literal": 0
      },
      "Operator": "+",
      "Right": {
        "Function": {
          "BlockStatement": [
            {
              "Return": null
            }
          ]
        }
      }
    }
  }
]
        `)

		test(`
        xyzzy // Ignore it
        // Ignore this
        // And this
        /* And all..



        ... of this!
        */
        "Nothing happens."
        // And finally this
        ---
[
  {
    "Identifier": "xyzzy"
  },
  {
    "Literal": "\"Nothing happens.\""
  }
]
        `)

		test(`
        ((x & (x = 1)) !== 0)
        ---
[
  {
    "BinaryExpression": {
      "Left": {
        "BinaryExpression": {
          "Left": {
            "Identifier": "x"
          },
          "Operator": "\u0026",
          "Right": {
            "Assign": {
              "Left": {
                "Identifier": "x"
              },
              "Right": {
                "Literal": 1
              }
            }
          }
        }
      },
      "Operator": "!==",
      "Right": {
        "Literal": 0
      }
    }
  }
]
        `)

		test(`
        { abc: 'def' }
        ---
[
  {
    "BlockStatement": [
      {
        "Label": {
          "Name": "abc",
          "Statement": {
            "Literal": "'def'"
          }
        }
      }
    ]
  }
]
        `)

		test(`
        // This is not an object, this is a string literal with a label!
        ({ abc: 'def' })
        ---
[
  {
    "Object": [
      {
        "Key": "abc",
        "Value": {
          "Literal": "'def'"
        }
      }
    ]
  }
]
        `)

		test(`
        [,]
        ---
[
  {
    "Array": [
      null
    ]
  }
]
        `)

		test(`
        [,,]
        ---
[
  {
    "Array": [
      null,
      null
    ]
  }
]
        `)

		test(`
        ({ get abc() {} })
        ---
[
  {
    "Object": [
      {
        "Key": "abc",
        "Value": {
          "Function": {
            "BlockStatement": []
          }
        }
      }
    ]
  }
]
        `)

		test(`
        /abc/.source
        ---
[
  {
    "Dot": {
      "Left": {
        "Literal": "/abc/"
      },
      "Member": "source"
    }
  }
]
        `)

		test(`
                xyzzy

        throw new TypeError("Nothing happens.")
        ---
[
  {
    "Identifier": "xyzzy"
  },
  {
    "Throw": {
      "New": {
        "ArgumentList": [
          {
            "Literal": "\"Nothing happens.\""
          }
        ],
        "Callee": {
          "Identifier": "TypeError"
        }
      }
    }
  }
]
	`)

		// When run, this will call a type error to be thrown
		// This is essentially the same as:
		//
		// var abc = 1(function(){})()
		//
		test(`
        var abc = 1
        (function(){
        })()
        ---
[
  {
    "Var": [
      [
        "abc",
        {
          "Call": {
            "ArgumentList": [],
            "Callee": {
              "Call": {
                "ArgumentList": [
                  {
                    "Function": {
                      "BlockStatement": []
                    }
                  }
                ],
                "Callee": {
                  "Literal": 1
                }
              }
            }
          }
        }
      ]
    ]
  }
]
        `)

		test(`
        "use strict"
        ---
[
  {
    "Literal": "\"use strict\""
  }
]
        `)

		test(`
        "use strict"
        abc = 1 + 2 + 11
        ---
[
  {
    "Literal": "\"use strict\""
  },
  {
    "Assign": {
      "Left": {
        "Identifier": "abc"
      },
      "Right": {
        "BinaryExpression": {
          "Left": {
            "BinaryExpression": {
              "Left": {
                "Literal": 1
              },
              "Operator": "+",
              "Right": {
                "Literal": 2
              }
            }
          },
          "Operator": "+",
          "Right": {
            "Literal": 11
          }
        }
      }
    }
  }
]
        `)

		test(`
        abc = function() { 'use strict' }
        ---
[
  {
    "Assign": {
      "Left": {
        "Identifier": "abc"
      },
      "Right": {
        "Function": {
          "BlockStatement": [
            {
              "Literal": "'use strict'"
            }
          ]
        }
      }
    }
  }
]
        `)

		test(`
        for (var abc in def) {
        }
        ---
[
  {
    "ForIn": {
      "Body": {
        "BlockStatement": []
      },
      "Into": [
        "abc",
        null
      ],
      "Source": {
        "Identifier": "def"
      }
    }
  }
]
        `)

		test(`
        abc = {
            '"': "'",
            "'": '"',
        }
        ---
[
  {
    "Assign": {
      "Left": {
        "Identifier": "abc"
      },
      "Right": {
        "Object": [
          {
            "Key": "\"",
            "Value": {
              "Literal": "\"'\""
            }
          },
          {
            "Key": "'",
            "Value": {
              "Literal": "'\"'"
            }
          }
        ]
      }
    }
  }
]
            `)

		return

		test(`
        if (!abc && abc.jkl(def) && abc[0] === +abc[0] && abc.length < ghi) {
        }
        ---
[
  {
    "If": {
      "Consequent": {
        "BlockStatement": []
      },
      "Test": {
        "BinaryExpression": {
          "Left": {
            "BinaryExpression": {
              "Left": {
                "BinaryExpression": {
                  "Left": null,
                  "Operator": "\u0026\u0026",
                  "Right": {
                    "Call": {
                      "ArgumentList": [
                        {
                          "Identifier": "def"
                        }
                      ],
                      "Callee": {
                        "Dot": {
                          "Left": {
                            "Identifier": "abc"
                          },
                          "Member": "jkl"
                        }
                      }
                    }
                  }
                }
              },
              "Operator": "\u0026\u0026",
              "Right": {
                "BinaryExpression": {
                  "Left": null,
                  "Operator": "===",
                  "Right": null
                }
              }
            }
          },
          "Operator": "\u0026\u0026",
          "Right": {
            "BinaryExpression": {
              "Left": {
                "Dot": {
                  "Left": {
                    "Identifier": "abc"
                  },
                  "Member": "length"
                }
              },
              "Operator": "\u003c",
              "Right": {
                "Identifier": "ghi"
              }
            }
          }
        }
      }
    }
  }
]
        `)
	})
}
