package parser

import (
	"github.com/corani/gotodo/gotodo"
)

type language string

const (
	// TODO(daniel) Support other languages besides go?
	GoLang language = "golang"
)

type Parser interface {
	Parse(config *gotodo.Config, path string) ([]gotodo.Comment, error)
}

func NewParserFor(lang language) Parser {
	return &GolangParser{}
}
