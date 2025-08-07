package main

import (
	"embed"

	"github.com/aborroy/alf-cli/cmd/alfresco"
)

//go:embed templates/** templates/.*.tmpl
var templateFS embed.FS

func main() {
	alfresco.TemplateFS = templateFS
	alfresco.Execute()
}
