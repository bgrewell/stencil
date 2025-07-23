package pkg

type PositionalArg struct {
	Name         string
	Description  string
	Required     bool
	DefaultValue string
	Choices      []string
}

type KeywordArg struct {
	Name         string
	Description  string
	Required     bool
	DefaultValue string
	Choices      []string
}
