//go:build ignore

// generate_catalog extracts and deduplicates the emoji sequences recognized by
// ebookgen's Markdown parser. Run it from the repository root as documented in
// _examples/README.md.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strconv"
	"strings"
)

func main() {
	source := flag.String("source", "", "path to ebookgen/internal/markdown/zemoji.go")
	output := flag.String("output", "catalog.txt", "output catalog")
	flag.Parse()
	if strings.TrimSpace(*source) == "" {
		fatalf("-source is required")
	}

	file, err := parser.ParseFile(token.NewFileSet(), *source, nil, 0)
	if err != nil {
		fatalf("parse source: %v", err)
	}
	entries := map[string]entry{}
	ast.Inspect(file, func(node ast.Node) bool {
		literal, ok := node.(*ast.CompositeLit)
		if !ok {
			return true
		}
		for _, element := range literal.Elts {
			pair, ok := element.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, keyOK := stringLiteral(pair.Key)
			value, valueOK := stringLiteral(pair.Value)
			if !keyOK || !valueOK || value == "" {
				continue
			}
			filename := emojiFilename(value)
			if previous, exists := entries[filename]; !exists || len(key) < len(previous.name) {
				entries[filename] = entry{name: key, sequence: value, filename: filename}
			}
		}
		return false
	})

	ordered := make([]entry, 0, len(entries))
	for _, item := range entries {
		ordered = append(ordered, item)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].filename < ordered[j].filename })

	fileOut, err := os.Create(*output)
	if err != nil {
		fatalf("create output: %v", err)
	}
	w := bufio.NewWriter(fileOut)
	for _, item := range ordered {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\n", item.filename, item.name, item.sequence); err != nil {
			fatalf("write output: %v", err)
		}
	}
	if err := w.Flush(); err != nil {
		fatalf("flush output: %v", err)
	}
	if err := fileOut.Close(); err != nil {
		fatalf("close output: %v", err)
	}
	fmt.Printf("generated %s with %d unique emoji sequences\n", *output, len(ordered))
}

type entry struct {
	filename string
	name     string
	sequence string
}

func stringLiteral(expr ast.Expr) (string, bool) {
	literal, ok := expr.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(literal.Value)
	return value, err == nil
}

func emojiFilename(cluster string) string {
	var codepoints []string
	for _, r := range cluster {
		if r != '\ufe0f' && r != '\ufe0e' {
			codepoints = append(codepoints, fmt.Sprintf("%x", r))
		}
	}
	return strings.Join(codepoints, "-")
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
