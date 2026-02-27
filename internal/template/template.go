package template

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/mocyuto/zgt/internal/config"
)

// Context holds information for template placeholders
type Context struct {
	Path          string
	Branch        string
	Repo          string // Main project root directory name
	CurrentDir    string // Current working directory name
	TargetBranch  string // Alias for Branch
	CurrentBranch string // Current branch name
}

func toHostname(s string) string {
	r := strings.NewReplacer("_", "-", "/", "-")
	return r.Replace(s)
}

// Replace replaces placeholders in the template string with values from the context.
// It uses text/template for substitution and supports custom functions.
// Available functions:
// - hostname: replaces '_' and '/' with '-'
func Replace(tmpl string, ctx Context) string {
	funcMap := template.FuncMap{
		"hostname": toHostname,
	}

	t, err := template.New("zgt").Funcs(funcMap).Parse(tmpl)
	if err != nil {
		// Fallback to original string if template parsing fails
		return tmpl
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, ctx); err != nil {
		return tmpl
	}

	return buf.String()
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

// ReplaceTmuxPanes replaces placeholders in the commands of each TmuxPane
func ReplaceTmuxPanes(panes []config.TmuxPane, ctx Context) []config.TmuxPane {
	res := make([]config.TmuxPane, len(panes))
	for i, p := range panes {
		res[i] = p
		res[i].Commands = ReplaceSlice(p.Commands, ctx)
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
	res.Tmux.Panes = ReplaceTmuxPanes(cfg.Tmux.Panes, ctx)
	res.Tmux.WindowName = Replace(cfg.Tmux.WindowName, ctx)
	return res
}
