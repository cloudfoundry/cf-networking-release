package styles

import "strings"

type StyleGroup struct {
	Styles map[string]Style
}

type Style struct {
	Tag         string
	SpecialChar string
}

func NewGroup() *StyleGroup {
	return &StyleGroup{
		Styles: map[string]Style{
			"reset": Style{Tag: "<RESET>", SpecialChar: "\x1B[0m"},
			"red":   Style{Tag: "<CLR_R>", SpecialChar: "\x1B[31;1m"},
			"green": Style{Tag: "<CLR_G>", SpecialChar: "\x1B[32;1m"},
			"cyan":  Style{Tag: "<CLR_C>", SpecialChar: "\x1B[36;1m"},
			"bold":  Style{Tag: "<BOLD>", SpecialChar: "\x1B[;1m"},
		},
	}
}

func (s *StyleGroup) ApplyStyles(body string) string {
	output := body
	for _, style := range s.Styles {
		output = strings.Replace(output, style.Tag, style.SpecialChar, -1)
	}
	return output
}

func (s *StyleGroup) AddStyle(body string, styleName string) string {
	style, ok := s.Styles[styleName]
	if !ok {
		return body
	}
	return style.Tag + body + s.Styles["reset"].Tag
}
