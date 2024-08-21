package utils

import (
	"bytes"
	"text/template"
)

var ParseTemplate = parseTemplate

func parseTemplate(tmpl string, data any) (string, error) {
	parsedTemplate, err := template.New("any").Parse(tmpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = parsedTemplate.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
