package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/corani/gotodo/parser"
	"github.com/fatih/color"
	"github.com/mitchellh/go-homedir"
)

/*
TODO(daniel) Don't use `log` for output, just `fmt.Fprintf` instead, based on config.Output.
*/

func loadConfig(configFile string) (*parser.Config, error) {
	contents, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read config file '%s': %v", configFile, err)
	}

	var config parser.Config
	err = json.Unmarshal(contents, &config)
	if err != nil {
		return nil, fmt.Errorf("unable to parse config file '%s': %v", configFile, err)
	}

	if config.RelRoot != "" {
		root, err := homedir.Expand(config.RelRoot)
		if err != nil {
			return nil, fmt.Errorf("unable to expand homedir: %v", err)
		}
		config.RelRoot = os.ExpandEnv(root)
	}

	var include []string
	for _, path := range config.Include {
		path, err := homedir.Expand(path)
		if err != nil {
			return nil, fmt.Errorf("unable to expand homedir: %v", err)
		}
		include = append(include, os.ExpandEnv(path))
	}
	config.Include = include

	return &config, nil
}

func main() {
	configFile := flag.String("config", "", "Path to config file")

	flag.Parse()
	if *configFile == "" {
		flag.Usage()
		return
	}

	config, err := loadConfig(*configFile)
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

	var comments []parser.Comment

	for _, path := range paths {
		// TODO(daniel) Support other languages besides go?
		p := parser.NewParserFor(parser.GoLang)
		c, err := p.Parse(config, path)
		if err != nil {
			log.Fatalf("%v", err)
		}
		comments = append(comments, c...)
	}

	comments = filterByAssignee(comments, config.Assignee, config.IncludeUnassigned)

	// TODO(daniel) Output formatters, "error", "json", "..."?
	switch config.Format {
	case "error":
		outputFormatError(config, comments)
	default:
		log.Printf("Unsupported formatter: '%s'\n", config.Format)
	}
}

func filterByAssignee(comments []parser.Comment, assignee string, includeUnassigned bool) []parser.Comment {
	var result []parser.Comment

	// NOTE(daniel) Check for assignee needs to be case-insensitive. Assignee in comment has been
	// lower-cased already.
	assignee = strings.ToLower(assignee)

	for _, comment := range comments {
		// NOTE(daniel) If an Assignee has been set, skip all comments that have a different
		// assignee (but include ones without an assignee, if requested).
		if assignee == "" {
			result = append(result, comment)
		} else if comment.Assignee == "" {
			if includeUnassigned {
				result = append(result, comment)
			}
		} else if comment.Assignee == assignee {
			result = append(result, comment)
		}
	}

	return result
}

func outputFormatError(config *parser.Config, comments []parser.Comment) {
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
