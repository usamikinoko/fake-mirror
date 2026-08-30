// Package plugins is a compatibility facade for the public extension API.
// New extensions should import rainhush/pkg/extension directly.
package plugins

import "rainhush/pkg/extension"

type Context = extension.Context
type Plugin = extension.Extension
type Fence = extension.Fence
type Assets = extension.Assets

func Register(p Plugin) { extension.Register(p, 200) }
func RenderFence(lang, content string, ctx Context) (string, bool) {
	return extension.Default.RenderFence(lang, content, ctx)
}
func CSS() [][]byte   { return extension.Default.CSS() }
func JS() [][]byte    { return extension.Default.JS() }
func Names() []string { return extension.Default.Names() }
func String() string  { return extension.Default.String() }
