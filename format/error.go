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
			if i > f.config.ContextLines {
				break
			}
			nocol.Println("\t" + line)
		}
		nocol.Println()
	}
}
