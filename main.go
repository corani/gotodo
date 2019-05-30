package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fatih/color"
	"github.com/mitchellh/go-homedir"
)

/*
TODO(daniel) Don't use `log` for output, just `fmt.Fprintf` instead, based on config.Output.
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
		path = os.ExpandEnv(path)

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
		root = os.ExpandEnv(root)
	}

	var comments []Comment

	for _, path := range paths {
		fs := token.NewFileSet()
		f, err := parser.ParseFile(fs, path, nil, parser.ParseComments)
		if err != nil {
			log.Printf("unable to parse '%s': %v\n", path, err)
		}
		cmap := ast.NewCommentMap(fs, f, f.Comments)

		comment := Comment{}
		for _, cg := range f.Comments {
			if comment.Filename != "" {
				comments = append(comments, comment)
				comment = Comment{}
			}

			// NOTE(daniel) Find the context of the comment group
			var node ast.Node
			for cm_node, cm_cgs := range cmap {
				for _, cm_cg := range cm_cgs {
					if cm_cg == cg {
						node = cm_node
					}
				}
			}

			for _, c := range getCommentLines(fs, cg) {
				for _, pattern := range config.Patterns {
					if i := strings.Index(c.Text, pattern); i == 0 {
						if comment.Filename != "" {
							comments = append(comments, comment)
						}

						pos := fs.PositionFor(c.Slash, false)
						if root != "" {
							pos.Filename, _ = filepath.Rel(root, pos.Filename)
						}
						comment = Comment{
							Filename: pos.Filename,
							Line:     uint(pos.Line),
							Col:      uint(pos.Column),
							Type:     pattern,
						}

						// NOTE(daniel) Store the context of the comment group
						if node != nil {
							var buf bytes.Buffer
							if err := format.Node(&buf, fs, node); err == nil {
								lines := strings.Split(buf.String(), "\n")

								// TODO(daniel) Sometimes the node that's found actually contains the comment-group.
								// This leads to duplicate output, so we need to find a way to avoid that.
								comment.Context = stripComments(lines)
							}
						}

						re := regexp.MustCompile(pattern + `\(([^\)]+)\)(.*)`)
						match := re.FindSubmatch([]byte(c.Text))
						if match != nil {
							comment.Assignee = strings.ToLower(string(match[1]))
							c.Text = string(match[2])
						} else {
							c.Text = c.Text[len(pattern):]
						}
						c.Text = strings.TrimPrefix(c.Text, ":")
					}
				}

				if comment.Filename != "" {
					comment.Text = append(comment.Text, strings.TrimSpace(c.Text))
				}
			}
		}
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

func stripComments(lines []string) []string {
	var result []string

	blockComment := false
	for _, line := range lines {
		if blockComment {
			if end := strings.Index(line, "*/"); end >= 0 {
				line = line[end+2:]
				blockComment = false
			} else {
				continue
			}
		} else {
			if start := strings.Index(line, "/*"); start >= 0 {
				line = line[:start]
				blockComment = true
			}
		}
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue
		}
		result = append(result, line)
	}
	return result
}

func filterByAssignee(comments []Comment, assignee string, includeUnassigned bool) []Comment {
	var result []Comment

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

func outputFormatError(config Config, comments []Comment) {
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
	}
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
