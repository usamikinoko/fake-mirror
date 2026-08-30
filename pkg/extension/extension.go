// Package extension defines the public compile-time extension API for Rainhush.
package extension

import (
	"html"
	"log"
	"sort"
	"strings"
)

// Context describes the document currently being rendered.
type Context struct {
	Document string
}

// Fence handles a fenced Markdown block. Return handled=false when the fence
// does not belong to the extension.
type Fence interface {
	RenderFence(lang string, content string, ctx Context) (output string, handled bool, err error)
}

// Assets contributes optional site-wide CSS and JavaScript to the build.
type Assets interface {
	CSS() [][]byte
	JS() [][]byte
}

// Extension is the minimum contract for a registered extension.
type Extension interface {
	Name() string
}

type entry struct {
	Extension
	Fence
	Assets
	priority int
}

// Registry stores extensions in deterministic priority order.
type Registry struct {
	entries []entry
	enabled map[string]bool
}

// NewRegistry creates an isolated registry, useful for custom builds and tests.
func NewRegistry() *Registry { return &Registry{enabled: make(map[string]bool)} }

// Default is the registry used by the standard Rainhush binary.
var Default = NewRegistry()

// Register adds an extension to the default registry. Extensions with lower
// priority run first; equal priorities retain registration order.
func Register(ext Extension, priority int) {
	Default.Register(ext, priority)
}

// SetEnabled changes an extension's runtime state for the default registry.
func SetEnabled(name string, enabled bool) { Default.SetEnabled(name, enabled) }

// SetEnabled changes an extension's runtime state. Unknown names are ignored,
// allowing configuration to be shared by binaries with different extensions.
func (r *Registry) SetEnabled(name string, enabled bool) {
	if r.enabled == nil {
		r.enabled = make(map[string]bool)
	}
	r.enabled[strings.TrimSpace(name)] = enabled
}

func (r *Registry) Register(ext Extension, priority int) {
	if ext == nil || strings.TrimSpace(ext.Name()) == "" {
		panic("rainhush: extension must have a name")
	}
	for _, item := range r.entries {
		if item.Name() == ext.Name() {
			panic("rainhush: duplicate extension name " + ext.Name())
		}
	}
	item := entry{Extension: ext, priority: priority}
	if fence, ok := ext.(Fence); ok {
		item.Fence = fence
	}
	if assets, ok := ext.(Assets); ok {
		item.Assets = assets
	}
	r.entries = append(r.entries, item)
	sort.SliceStable(r.entries, func(i, j int) bool {
		return r.entries[i].priority < r.entries[j].priority
	})
}

// RenderFence dispatches a block to extensions in priority order.
func (r *Registry) RenderFence(lang, content string, ctx Context) (string, bool) {
	for _, item := range r.entries {
		if !r.isEnabled(item.Name()) || item.Fence == nil {
			continue
		}
		out, handled, err := item.Fence.RenderFence(lang, content, ctx)
		if err != nil {
			log.Printf("[extension] %s: %s fence %q error: %v", ctx.Document, item.Name(), lang, err)
			return `<div class="extension-error">Extension error rendering "` + html.EscapeString(lang) + `": ` + html.EscapeString(err.Error()) + `</div>`, true
		}
		if handled {
			return out, true
		}
	}
	return "", false
}

// CSS returns all registered extension styles in deterministic order.
func (r *Registry) CSS() [][]byte {
	var out [][]byte
	for _, item := range r.entries {
		if r.isEnabled(item.Name()) && item.Assets != nil {
			out = append(out, item.Assets.CSS()...)
		}
	}
	return out
}

// JS returns all registered extension scripts in deterministic order.
func (r *Registry) JS() [][]byte {
	var out [][]byte
	for _, item := range r.entries {
		if r.isEnabled(item.Name()) && item.Assets != nil {
			out = append(out, item.Assets.JS()...)
		}
	}
	return out
}

func (r *Registry) isEnabled(name string) bool {
	enabled, exists := r.enabled[name]
	return !exists || enabled
}

// Names returns registered extension names in registration priority order.
func (r *Registry) Names() []string {
	out := make([]string, 0, len(r.entries))
	for _, item := range r.entries {
		out = append(out, item.Name())
	}
	return out
}

// String returns a readable extension summary.
func (r *Registry) String() string { return strings.Join(r.Names(), ", ") }
