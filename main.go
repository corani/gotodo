package main

import (
	"flag"
	"log"
	"os"

	"github.com/corani/gotodo/format"
	"github.com/corani/gotodo/glob"
	"github.com/corani/gotodo/gotodo"
	"github.com/corani/gotodo/parser"
)

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

	paths := glob.Glob(config)

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

	formatter := format.NewFormatter(config)
	formatter.Format(comments)

	// TODO(daniel) Make the return code configurable based on the pattern?
	rc := 0
loop:
	for _, comment := range comments {
		switch comment.Type {
		case "TODO":
			rc = 1
		case "FIXME":
			rc = 2
			break loop
		}
	}
	os.Exit(rc)
}
