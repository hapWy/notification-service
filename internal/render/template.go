// Package render fills a notification template body with the params
// supplied on the request.
package render

import (
	"bytes"
	"fmt"
	"text/template"
)

// Render executes body as a Go text/template against params (e.g. a
// template body of "Hello {{.name}}!" with params {"name": "Alice"}
// produces "Hello Alice!"). Missing keys render as the zero value rather
// than failing, so a template referencing an optional param degrades
// gracefully instead of blocking delivery.
func Render(body string, params map[string]interface{}) (string, error) {
	tmpl, err := template.New("notification").Option("missingkey=zero").Parse(body)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, params); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return buf.String(), nil
}
