package format

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/corani/gotodo/gotodo"
)

type InfraboxFormatter struct {
	config *gotodo.Config
}

type data struct {
	Label string `json:"label"`
	Value int    `json:"value"`
	Color string `json:"color"`
}

type element struct {
	Type     string      `json:"type"`
	Rows     [][]element `json:"rows,omitempty"`
	Headers  []element   `json:"headers,omitempty"`
	Text     string      `json:"text,omitempty"`
	Color    string      `json:"color,omitempty"`
	Emphasis string      `json:"emphasis,omitempty"`
	Name     string      `json:"name,omitempty"`
	Data     []data      `json:"data,omitempty"`
}

type markup struct {
	Version  int       `json:"version"`
	Title    string    `json:"title"`
	Elements []element `json:"elements,omitempty"`
}

func (f *InfraboxFormatter) Format(comments []gotodo.Comment) {
	// TODO(daniel) These shouldn't be hardcoded!
	output := map[string][]gotodo.Comment{
		"FIXME": []gotodo.Comment{},
		"TODO":  []gotodo.Comment{},
		"NOTE":  []gotodo.Comment{},
	}

	for _, comment := range comments {
		output[comment.Type] = append(output[comment.Type], comment)
	}

	infra := markup{
		Version: 1,
		Title:   "Todo",
		Elements: []element{
			element{
				Type: "h1",
				Text: fmt.Sprintf("FIXME: %02d / TODO: %02d / NOTE: %02d", len(output["FIXME"]), len(output["TODO"]), len(output["NOTE"])),
			},
		},
	}

	for tag, comments := range output {
		count := len(comments)
		if count > 0 {
			rows := [][]element{}
			for _, comment := range comments {
				if comment.Assignee == "" {
					comment.Assignee = "-"
				}

				rows = append(rows, []element{
					element{
						Type: "text",
						Text: fmt.Sprintf("%s:%d:%d", comment.Filename, comment.Line, comment.Col),
					},
					element{
						Type: "text",
						Text: comment.Assignee,
					},
					element{
						Type: "text",
						Text: strings.Join(comment.Text, "<br/>\n"),
					},
					// TODO(daniel) Consider config.ContextLines
					element{
						Type: "text",
						Text: "<pre>" + strings.Join(comment.Context, "\n") + "</pre>",
					},
				})
			}

			infra.Elements = append(infra.Elements, element{
				Type: "h1",
				Text: fmt.Sprintf("%s (%d)", tag, count),
			})
			infra.Elements = append(infra.Elements, element{
				Type: "table",
				Headers: []element{
					element{
						Type: "text",
						Text: "Location",
					},
					element{
						Type: "text",
						Text: "Assignee",
					},
					element{
						Type: "text",
						Text: "Text",
					},
					element{
						Type: "text",
						Text: "Context",
					},
				},
				Rows: rows,
			})

		}
	}

	out := getOutputStream(f.config)
	defer out.Close()

	// NOTE(daniel) By default, json.Marshal will escape HTML, which is not what we want. Use the encoder and disable this behavior.
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "\t")
	enc.Encode(infra)
}
