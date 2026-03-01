package email

import (
	"html/template"

	rendermarkdown "github.com/bakkerme/curator-ai/internal/render/markdown"
)

// TemplateFuncs returns the built-in helper functions available to email templates.
func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"toHTML": func(input string) (template.HTML, error) {
			rendered, err := rendermarkdown.Render(input)
			if err != nil {
				return "", err
			}
			return template.HTML(rendered), nil
		},
	}
}
