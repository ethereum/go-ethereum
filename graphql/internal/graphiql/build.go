package graphiql

import (
	"embed"
	"io"

	"github.com/ethereum/go-ethereum/log"
)

//go:embed *.js *.css *.html
var files embed.FS

var Assets map[string][]byte

func init() {
	Assets = make(map[string][]byte)
	names := []string{"index.html", "graphiql.min.css", "graphiql.min.js", "react.production.min.js", "react-dom.production.min.js"}
	for _, name := range names {
		f, err := files.Open(name)
		if err != nil {
			log.Warn("failed to load graphiql asset", "asset", name, "err", err)
		}
		a, err := io.ReadAll(f)
		if err != nil {
			log.Warn("failed to read graphiql asset", "asset", name, "err", err)
		}
		Assets[name] = a
	}
}
