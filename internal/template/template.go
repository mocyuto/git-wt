package template

import (
	"strings"

	"github.com/mocyuto/git-wt/internal/config"
)

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

// ReplaceMap replaces placeholders in all values of the map with values from the context
func ReplaceMap(m map[string]string, ctx Context) map[string]string {
	res := make(map[string]string, len(m))
	for k, v := range m {
		res[k] = Replace(v, ctx)
	}
	return res
}

// ReplaceSlice replaces placeholders in all elements of the slice with values from the context
func ReplaceSlice(s []string, ctx Context) []string {
	res := make([]string, len(s))
	for i, v := range s {
		res[i] = Replace(v, ctx)
	}
	return res
}

// ReplaceConfig returns a copy of the config with placeholders replaced
func ReplaceConfig(cfg config.Config, ctx Context) config.Config {
	res := cfg
	res.Hooks.Add = ReplaceSlice(cfg.Hooks.Add, ctx)
	res.Hooks.RM = ReplaceSlice(cfg.Hooks.RM, ctx)
	res.Ignore = ReplaceSlice(cfg.Ignore, ctx)
	res.Env = ReplaceMap(cfg.Env, ctx)
	return res
}
