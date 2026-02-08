package template

import "strings"

// Context holds information for template placeholders
type Context struct {
	Path   string
	Branch string
	Repo   string
}

// Replace replaces placeholders in the template string with values from the context
func Replace(tmpl string, ctx Context) string {
	replaced := strings.ReplaceAll(tmpl, "{{.Path}}", ctx.Path)
	replaced = strings.ReplaceAll(replaced, "{{.Branch}}", ctx.Branch)
	replaced = strings.ReplaceAll(replaced, "{{.Repo}}", ctx.Repo)
	return replaced
}
