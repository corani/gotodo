package format

import (
	"io"
	"log"
	"os"
	"strings"

	"github.com/corani/gotodo/gotodo"
)

type Formatter interface {
	Format([]gotodo.Comment)
}

func NewFormatter(config *gotodo.Config) Formatter {
	// TODO(daniel) Add a formatter for XUnit
	switch config.Format {
	case "error":
		return &ErrorFormatter{config: config}
	case "json":
		return &JsonFormatter{config: config}
	case "xunit":
		log.Fatalf("Unsupported formatter: '%s'\n", config.Format)
	case "infrabox":
		return &InfraboxFormatter{config: config}
	default:
		log.Fatalf("Unsupported formatter: '%s'\n", config.Format)
	}
	return nil
}

// NOTE(daniel) We're returning STDOUT/STDERR as if they were user-created files.
// To avoid closing them, wrap them and ignore the `Close` call.
type writerIgnoreCloser struct {
	delegate io.Writer
}

func (w writerIgnoreCloser) Write(p []byte) (n int, err error) {
	return w.delegate.Write(p)
}

func (w writerIgnoreCloser) Close() error {
	return nil
}

func ignoreClose(w io.Writer) io.WriteCloser {
	return writerIgnoreCloser{w}
}

func getOutputStream(config *gotodo.Config) io.WriteCloser {
	switch {
	case config.Output == "":
		fallthrough
	case strings.ToUpper(config.Output) == "STDOUT":
		return ignoreClose(os.Stdout)
	case strings.ToUpper(config.Output) == "STDERR":
		return ignoreClose(os.Stderr)
	default:
		file, err := os.Create(config.Output)
		if err != nil {
			log.Fatalf("Unable to create output file: '%s'\n", config.Output)
		}
		return file
	}
}
