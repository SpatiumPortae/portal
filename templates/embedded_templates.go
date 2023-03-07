package templates

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"strings"
)

const (
	templatesDir      = "tmpl"
	templateExtension = ".html"
)

//go:embed tmpl
var embeddedFiles embed.FS

// Parse template entries from templatesDir, mapped to filenames without the prefixed templatesDir.
// i.e. templates/tmpl/relay/landing.html ---> map["relay/landing.html" -> *template.Template].
func NewTemplates() (map[string]*template.Template, error) {
	templates := make(map[string]*template.Template)
	err := fs.WalkDir(embeddedFiles, templatesDir, func(path string, _ fs.DirEntry, err error) error {
		if strings.HasSuffix(path, templateExtension) {
			tmpl, err := template.ParseFS(embeddedFiles, path)
			if err != nil {
				return err
			}
			templates[strings.TrimPrefix(path, templatesDir+"/")] = tmpl
		}
		return err
	})
	if err != nil {
		return templates, fmt.Errorf("parsing template files: %w", err)
	}
	return templates, nil
}
