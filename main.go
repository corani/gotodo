package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/corani/gotodo/gotodo"
	"github.com/corani/gotodo/parser"
	"github.com/fatih/color"
)

/*
TODO(daniel) Don't use `log` for output, just `fmt.Fprintf` instead, based on config.Output.
*/

func main() {
	configFile := flag.String("config", "", "Path to config file")

	flag.Parse()
	if *configFile == "" {
		flag.Usage()
		return
	}

	config, err := gotodo.LoadConfig(*configFile)
	if err != nil {
		log.Fatal(err)
	}

	var paths []string
	for _, path := range config.Include {
		// TODO(daniel) Use include/exclude patterns here. Since the Go standard library doesn't support
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

	var comments []gotodo.Comment

	for _, path := range paths {
		// TODO(daniel) Support other languages besides go?
		p := parser.NewParserFor(parser.GoLang)
		c, err := p.Parse(config, path)
		if err != nil {
			log.Fatalf("%v", err)
		}
		comments = append(comments, c...)
	}

	comments = config.FilterByAssignee(comments)

	// TODO(daniel) Output formatters, "error", "json", "..."?
	switch config.Format {
	case "error":
		outputFormatError(config, comments)
	default:
		log.Printf("Unsupported formatter: '%s'\n", config.Format)
	}
}

func outputFormatError(config *gotodo.Config, comments []gotodo.Comment) {
	// TODO(daniel) make colors dynamic
	bold := color.New(color.Bold).SprintFunc()
	underline := color.New(color.Underline).SprintFunc()
	nocol := color.New(color.FgWhite)
	for _, comment := range comments {
		col := nocol
		switch comment.Type {
		case "NOTE":
			col = color.New(color.FgHiGreen)
		case "TODO":
			col = color.New(color.FgHiYellow)
		case "FIXME":
			col = color.New(color.FgHiRed)
		}

		tag := comment.Type
		if comment.Assignee != "" {
			tag = tag + "(" + underline(comment.Assignee) + ")"
		}
		col.Printf("%s:%d:%d %s\n", comment.Filename, comment.Line, comment.Col, bold(tag))
		for _, line := range comment.Text {
			col.Println("\t// " + line)
		}
		for i, line := range comment.Context {
			if i > config.ContextLines {
				break
			}
			nocol.Println("\t" + line)
		}
		nocol.Println()
	}
}
