package parser

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

type language string

const (
	GoLang language = "golang"
)

type Parser interface {
	Parse(config *Config, path string) ([]Comment, error)
}

func NewParserFor(lang language) Parser {
	return &GolangParser{}
}
