package main

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/swarm"
	"github.com/ethereum/go-ethereum/swarm/api"
)

var port = 8500

func bzzREPL(t *testing.T, configf func(*api.Config)) (string, string, *testjethre, *node.Node) {
	prvKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal("unable to generate key")
	}
	bzztmp, err := ioutil.TempDir("", "bzz-js-test")
	config, err := api.NewConfig(bzztmp, common.Address{}, prvKey)
	if err != nil {
		t.Fatal("unable to configure swarm")
	}
	if configf != nil {
		configf(config)
	}
	tmp, repl, stack := testREPL(t, func(n *node.Node) {
		if err := n.Register(func(ctx *node.ServiceContext) (node.Service, error) {
			return swarm.NewSwarm(ctx, config, false)
		}); err != nil {
			t.Fatalf("Failed to register the Swarm service: %v", err)
		}
	})
	return bzztmp, tmp, repl, stack
}

func withREPL(t *testing.T, cf func(*api.Config), f func(repl *testjethre)) {
	bzztmp, tmp, repl, stack := bzzREPL(t, cf)
	defer stack.Stop()
	defer os.RemoveAll(tmp)
	defer os.RemoveAll(bzztmp)
	f(repl)
}

func TestBzzPutGet(t *testing.T) {
	withREPL(t,
		func(c *api.Config) {
			c.Port = ""
		}, func(repl *testjethre) {
			if checkEvalJSON(t, repl, `hash = bzz.put("console.log(\"hello from console\")", "application/javascript")`, `"97f1b7c7ea12468fd37c262383b9aa862d0cfbc4fc7218652374679fc5cf40cd"`) != nil {
				return
			}
			want := `{"content":"console.log(\"hello from console\")","contentType":"application/javascript","size":"33","status":"0"}`
			if checkEvalJSON(t, repl, `bzz.get(hash)`, want) != nil {
				return
			}
		})
}

// the server can be initialized only once per test session !
// until we implement a stoppable http server
// further http tests will need to make sure the correct server is running
func TestHTTP(t *testing.T) {
	withREPL(t, nil, func(repl *testjethre) {
		if checkEvalJSON(t, repl, `hash = bzz.put("f42 = function() { return 42 }", "application/javascript")`, `"e6847876f00102441f850b2d438a06d10e3bf24e6a0a76d47b073a86c3c2f9ac"`) != nil {
			return
		}
		if checkEvalJSON(t, repl, `admin.httpGet("bzz://"+hash)`, `"f42 = function() { return 42 }"`) != nil {
			return
		}

		// if checkEvalJSON(t, repl, `http.loadScript("bzz://"+hash)`, `true`) != nil {
		// 	return
		// }

		// if checkEvalJSON(t, repl, `f42()`, `42`) != nil {
		// 	return
		// }
	})
}
