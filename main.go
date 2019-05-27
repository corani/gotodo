package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/mitchellh/go-homedir"
)

/*
TODO(daniel) Don't use `log` for output, just `fmt.Printf` instead.
*/
type Config struct {
	Patterns          []string `json:"patterns"`
	Assignee          string   `json:"assignee"`
	IncludeUnassigned bool     `json:"includeUnassigned"`
	Include           []string `json:"include"`
	Exclude           []string `json:"exclude"`
	Format            string   `json:"format"`
	Output            string   `json:"output"`
	RelRoot           string   `json:"relRoot"`
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

	var paths []string

	for _, path := range config.Include {
		path, err := homedir.Expand(path)
		if err != nil {
			log.Fatalf("unable to expand homedir: %v", err)
		}

		// TODO(daniel): Use include/exclude patterns here. Since the Go standard library doesn't support
		// double-star globs, we need to write our own matcher here.
		filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.Mode().IsRegular() && strings.HasSuffix(info.Name(), ".go") {
				paths = append(paths, path)
			}
			return nil
		})
	}

	var root string
	if config.RelRoot != "" {
		root, err = homedir.Expand(config.RelRoot)
		if err != nil {
			log.Fatalf("unable to expand homedir: %v", err)
		}
	}

	for _, path := range paths {
		fs := token.NewFileSet()
		ast, err := parser.ParseFile(fs, path, nil, parser.ParseComments)
		if err != nil {
			log.Printf("unable to parse '%s': %v\n", path, err)
		}

		for _, cg := range ast.Comments {
			found := ""
			first := true
			comments := getCommentLines(fs, cg)
		comment:
			for _, c := range comments {
				for _, pattern := range config.Patterns {
					if pos := strings.Index(c.Text, pattern); pos == 0 {
						// NOTE(daniel) If an Assignee has been set, skip all comments that have a different
						// assignee (but include ones without an assignee, if requested).
						if !strings.HasPrefix(c.Text, pattern+"(") {
							if !config.IncludeUnassigned {
								continue comment
							}
						} else if config.Assignee != "" {
							if !strings.HasPrefix(c.Text, fmt.Sprintf("%s(%s)", pattern, config.Assignee)) {
								continue comment
							}
						}
						found = pattern
						first = true
					}
				}
				// TODO(daniel): Is there some way to get the context of the comment group? It would be nice if
				// we could print a few code-lines, as the comments might not always make sense otherwise
				// TODO(daniel): Do we need some kind of intermediate representation, before we output?
				// TODO(daniel) highlight the pattern
				// TODO(daniel) make colors dynamic
				if found != "" {
					var col *color.Color
					switch found {
					case "NOTE":
						col = color.New(color.FgHiGreen)
					case "TODO":
						col = color.New(color.FgHiYellow)
					case "FIXME":
						col = color.New(color.FgHiRed)
					}

					pos := fs.PositionFor(c.Slash, false)
					if root != "" {
						pos.Filename, _ = filepath.Rel(root, pos.Filename)
					}
					if first {
						first = false
						col.Printf("%s %s\n", pos, c.Text)
					} else {
						col.Printf("%*s %s\n", len(fmt.Sprintf("%s", pos)), "", c.Text)
					}
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
				// TODO(daniel): Increment the c.Slash position, so we land exactly on the pattern (skip e.g. "// ")
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
