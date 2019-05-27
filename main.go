package main

import (
	"encoding/json"
	"flag"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Patterns []string `json:"patterns"`
	Assignee string   `json:"assignee"`
	Include  []string `json:"include"`
	Exclude  []string `json:"exclude"`
	Format   string   `json:"format"`
	Output   string   `json:"output"`
}

func main() {
	configFile := flag.String("config", "", "Path to config file")

	flag.Parse()
	if *configFile == "" {
		flag.Usage()
		return
	}

	contents, err := ioutil.ReadFile(*configFile)
	if err != nil {
		log.Fatalf("unable to read config file '%s': %v", *configFile, err)
	}

	var config Config
	err = json.Unmarshal(contents, &config)
	if err != nil {
		log.Fatalf("unable to parse config file '%s': %v", *configFile, err)
	}

	// TODO(daniel): Use include/exclude patterns here. Since the Go standard library doesn't support
	// double-star globs, we need to write our own matcher here.
	var paths []string
	filepath.Walk("/home/dbos/go/src", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() && strings.HasSuffix(info.Name(), ".go") {
			paths = append(paths, path)
		}
		return nil
	})

	for _, path := range paths {
		fs := token.NewFileSet()
		ast, err := parser.ParseFile(fs, path, nil, parser.ParseComments)
		if err != nil {
			log.Printf("unable to parse '%s': %v\n", path, err)
		}

		for _, cg := range ast.Comments {
			found := false
			comments := getCommentLines(fs, cg)
			for _, c := range comments {
				// TODO(daniel): Support having more than one match within the same comment group
				if !found {
					for _, pattern := range config.Patterns {
						// TODO(daniel): Extract assignee
						if pos := strings.Index(c.Text, pattern); pos == 0 {
							found = true
						}
					}
				}
				// TODO(daniel): Is there some way to get the context of the comment group? It would be nice if
				// we could print a few code-lines, as the comments might not always make sense otherwise
				// TODO(daniel): Do we need some kind of intermediate representation, before we output?
				if found {
					pos := fs.PositionFor(c.Slash, false)
					log.Printf("%s %s", pos, c.Text)
				}
			}
		}
	}

	// TODO(daniel) Output formatters, "error", "json", "..."?
}

func getCommentLines(fs *token.FileSet, cg *ast.CommentGroup) []*ast.Comment {
	var result []*ast.Comment
	for _, c := range cg.List {
		// NOTE(daniel): Windows/Mac line endings (\r) will be removed later, so no need to consider them here
		parts := strings.Split(c.Text, "\n")
		file := fs.File(c.Slash)
		offset := file.Offset(c.Slash)
		for _, s := range parts {
			pos := file.Pos(offset)
			offset += len(s)
			if strings.HasPrefix(s, "//") || strings.HasPrefix(s, "/*") {
				s = s[2:]
			}
			if strings.HasSuffix(s, "*/") {
				s = s[:len(s)-2]
			}
			s := strings.Trim(s, " \r\t")

			result = append(result, &ast.Comment{
				Slash: pos,
				Text:  s,
			})
		}
	}
	return result
}
