package gotodo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
)

type Config struct {
	Patterns          []string `json:"patterns"`
	Assignee          string   `json:"assignee"`
	IncludeUnassigned bool     `json:"includeUnassigned"`
	Include           []string `json:"include"`
	Exclude           []string `json:"exclude"`
	Format            string   `json:"format"`
	Output            string   `json:"output"`
	RelRoot           string   `json:"relRoot"`
	ContextLines      int      `json:"contextLines"`
}

type Comment struct {
	Filename string
	Line     uint
	Col      uint
	Type     string
	Assignee string
	Text     []string
	Context  []string
}

func LoadConfig(configFile string) (*Config, error) {
	contents, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read config file '%s': %v", configFile, err)
	}

	var config Config
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

func (c *Config) FilterByAssignee(comments []Comment) []Comment {
	var result []Comment

	// NOTE(daniel) Check for assignee needs to be case-insensitive. Assignee in comment has been
	// lower-cased already.
	assignee := strings.ToLower(c.Assignee)

	for _, comment := range comments {
		// NOTE(daniel) If an Assignee has been set, skip all comments that have a different
		// assignee (but include ones without an assignee, if requested).
		if assignee == "" {
			result = append(result, comment)
		} else if comment.Assignee == "" {
			if c.IncludeUnassigned {
				result = append(result, comment)
			}
		} else if comment.Assignee == assignee {
			result = append(result, comment)
		}
	}

	return result
}
