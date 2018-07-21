package templating

import (
	"path/filepath"
	"os"
	"strings"
	"io/ioutil"
	"html/template"
	"github.com/microcosm-cc/bluemonday"
)

/// BuildDefaultFunctionMap builds a default map of functions common to all templates.
func BuildDefaultFunctionMap() template.FuncMap {
	return template.FuncMap{
		"toPlainText": toPlainText,
		"stripUnsafeTags": stripUnsafeTags,
	}
}

/// toPlainText removes any HTML tags from the given target string.
func toPlainText(target string) string {
	return bluemonday.StrictPolicy().Sanitize(target)
}

/// stripUnsafeTags strips any unsafe HTML tags from the given target string.
func stripUnsafeTags(target string) string {
	return bluemonday.UGCPolicy().Sanitize(target)
}

/// FindAndParseTemplates finds all templates in a directory `rootDir` and all sub directories.
func FindAndParseTemplates(rootDir string, funcMap template.FuncMap) (*template.Template, error) {
	cleanRoot := filepath.Clean(rootDir)
	pfx := len(cleanRoot)+1
	root := template.New("")

	err := filepath.Walk(cleanRoot, func(path string, info os.FileInfo, e1 error) error {
		if !info.IsDir() && strings.HasSuffix(path, ".html") || strings.HasSuffix(path, ".txt") {
			if e1 != nil {
				return e1
			}

			b, e2 := ioutil.ReadFile(path)
			if e2 != nil {
				return e2
			}

			name := path[pfx:]
			t := root.New(name).Funcs(funcMap)
			t, e2 = t.Parse(string(b))
			if e2 != nil {
				return e2
			}
		}

		return nil
	})

	return root, err
}