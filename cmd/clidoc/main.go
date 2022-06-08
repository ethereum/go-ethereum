package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/internal/cli"
)

const (
	DefaultDir      string = "./docs/cli"
	DefaultMainPage string = "README.md"
)

func main() {
	commands := cli.Commands()

	dest := flag.String("d", DefaultDir, "Destination directory where the docs will be generated")
	flag.Parse()

	dirPath := filepath.Join(".", *dest)
	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		log.Fatalln("Failed to create directory.", err)
	}

	mainPage := []string{
		"# Bor command line interface",
		"## Commands",
	}

	keys := make([]string, len(commands))
	i := 0

	for k := range commands {
		keys[i] = k
		i++
	}

	sort.Strings(keys)

	for _, name := range keys {
		cmd, err := commands[name]()
		if err != nil {
			log.Fatalf("Error occurred when inspecting bor command %s: %s", name, err)
		}

		fileName := strings.ReplaceAll(name, " ", "_") + ".md"

		overwriteFile(filepath.Join(dirPath, fileName), cmd.MarkDown())
		mainPage = append(mainPage, "- [```"+name+"```](./"+fileName+")")
	}

	overwriteFile(filepath.Join(dirPath, DefaultMainPage), strings.Join(mainPage, "\n\n"))

	os.Exit(0)
}

func overwriteFile(filePath string, text string) {
	log.Printf("Writing to page: %s\n", filePath)

	f, err := os.Create(filePath)
	if err != nil {
		log.Fatalln(err)
	}

	if _, err = f.WriteString(text); err != nil {
		log.Fatalln(err)
	}

	if err = f.Close(); err != nil {
		log.Fatalln(err)
	}
}
