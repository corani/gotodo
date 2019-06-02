package format

import (
	"encoding/json"
	"fmt"

	"github.com/corani/gotodo/gotodo"
)

type JsonFormatter struct {
	config *gotodo.Config
}

func (f *JsonFormatter) Format(comments []gotodo.Comment) {
	out := getOutputStream(f.config)
	defer out.Close()
	b, err := json.MarshalIndent(comments, "", "\t")
	if err == nil {
		fmt.Fprintln(out, string(b))
	}
}
