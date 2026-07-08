package agentstatus

import "embed"

// PluginAssets embeds the opencode status plugin so `zgt agent install`
// can ship it without touching the filesystem at build time.
//
//go:embed assets/zgt-status.js
var PluginAssets embed.FS

// OpenCodePluginName is the filename written into
// ~/.config/opencode/plugins/ on install.
const OpenCodePluginName = "zgt-status.js"

// OpenCodePluginSource returns the embedded opencode plugin source.
func OpenCodePluginSource() ([]byte, error) {
	return PluginAssets.ReadFile("assets/zgt-status.js")
}
