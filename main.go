package main

import (
	"flag"
	"log"

	"github.com/corani/gotodo/format"
	"github.com/corani/gotodo/glob"
	"github.com/corani/gotodo/gotodo"
	"github.com/corani/gotodo/parser"
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
}
