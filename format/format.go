package format

import (
	"log"

	"github.com/corani/gotodo/gotodo"
)

type Formatter interface {
	Format([]gotodo.Comment)
}

func NewFormatter(config *gotodo.Config) Formatter {
	// TODO(daniel) Output formatters, "error", "json", "..."?
	switch config.Format {
	case "error":
		return &ErrorFormatter{config: config}
	default:
		log.Fatalf("Unsupported formatter: '%s'\n", config.Format)
	}
	return nil
}
