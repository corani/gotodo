package glob

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/corani/gotodo/gotodo"
)

func Glob(config *gotodo.Config) []string {
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

	return paths
}
