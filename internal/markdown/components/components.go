// Package components is a compatibility facade for legacy built-in components.
// New built-ins and third-party extensions should use rainhush/pkg/extension.
package components

import "rainhush/pkg/extension"

// Component is the pre-extension built-in component contract.
type Component interface {
	Name() string
	Render(lang string, content string, document string) (html string, handled bool)
	CSS() [][]byte
	JS() [][]byte
}

type adapter struct{ Component }

func (a adapter) RenderFence(lang, content string, ctx extension.Context) (string, bool, error) {
	out, handled := a.Render(lang, content, ctx.Document)
	return out, handled, nil
}

// Register gives legacy built-ins precedence over third-party extensions.
func Register(c Component) { extension.Register(adapter{c}, 100) }
func RenderFence(lang, content, document string) (string, bool) {
	return extension.Default.RenderFence(lang, content, extension.Context{Document: document})
}
func CSS() [][]byte   { return extension.Default.CSS() }
func JS() [][]byte    { return extension.Default.JS() }
func Names() []string { return extension.Default.Names() }
func String() string  { return extension.Default.String() }
