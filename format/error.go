package format

import (
	"github.com/corani/gotodo/gotodo"
	"github.com/fatih/color"
)

type ErrorFormatter struct {
	config *gotodo.Config
}

func (f *ErrorFormatter) Format(comments []gotodo.Comment) {
	// TODO(daniel) make colors dynamic
	bold := color.New(color.Bold).SprintFunc()
	underline := color.New(color.Underline).SprintFunc()
	nocol := color.New(color.FgWhite)

	type summaryType struct {
		format *color.Color
		count  int
	}
	summary := map[string]summaryType{
		"NOTE": summaryType{
			format: color.New(color.FgHiGreen),
		},
		"TODO": summaryType{
			format: color.New(color.FgHiYellow),
		},
		"FIXME": summaryType{
			format: color.New(color.FgHiRed),
		},
	}
	for _, comment := range comments {
		entry := summary[comment.Type]
		entry.count++
		summary[comment.Type] = entry

		col := entry.format

		tag := comment.Type
		if comment.Assignee != "" {
			tag = tag + "(" + underline(comment.Assignee) + ")"
		}
		col.Printf("%s:%d:%d %s\n", comment.Filename, comment.Line, comment.Col, bold(tag))
		for _, line := range comment.Text {
			col.Println("\t// " + line)
		}
		for i, line := range comment.Context {
			if i > f.config.ContextLines {
				break
			}
			nocol.Println("\t" + line)
		}
		nocol.Println()
	}

	if len(summary) > 0 {
		nocol.Printf("Summary: ")
		separator := ""
		for k, v := range summary {
			nocol.Printf("%s", separator)
			v.format.Printf("%s: %s", k, bold(v.count))
			separator = " / "
		}
		nocol.Println()
	}
}
