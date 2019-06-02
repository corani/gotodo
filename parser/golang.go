package parser

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"path/filepath"
	"regexp"
	"strings"
)

type GolangParser struct{}

func (p *GolangParser) Parse(config *Config, path string) ([]Comment, error) {
	var comments []Comment
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
					if config.RelRoot != "" {
						pos.Filename, _ = filepath.Rel(config.RelRoot, pos.Filename)
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
	return comments, nil
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