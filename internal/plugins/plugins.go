package plugins

import (
	"html"
	"log"
	"sort"
	"strings"
)

// Context 是渲染钩子的调用上下文，携带当前渲染的文档名供插件诊断日志使用。
type Context struct {
	Document string
}

// Plugin 是所有插件的公共接口，注册的插件至少需要提供唯一名称。
type Plugin interface {
	Name() string
}

// Fence 是可选的围栏代码块渲染钩子。
// lang 是围栏语言（如 image-layout-d、mermaid、custom-xyz），content 是块内原始文本。
// 返回 ok=true 表示该插件接管渲染（输出 html）；false 表示不处理，交给后续插件或默认渲染器。
type Fence interface {
	RenderFence(lang string, content string, ctx Context) (html string, ok bool, err error)
}

// Assets 是可选的资源提供钩子，返回需要合并进站点全局 bundle 的 CSS / JS 内容。
type Assets interface {
	CSS() [][]byte
	JS() [][]byte
}

var (
	plugins []Plugin
	fences  []Fence
	assets  []Assets
)

// Register 注册一个插件，在插件包的 init() 中调用。
func Register(p Plugin) {
	plugins = append(plugins, p)
	if f, ok := p.(Fence); ok {
		fences = append(fences, f)
	}
	if a, ok := p.(Assets); ok {
		assets = append(assets, a)
	}
}

// RenderFence 按注册顺序把围栏代码块分发给各插件的 Fence 钩子。
// 首个声明接管（ok=true）的插件决定输出；插件返回错误时记录日志并输出错误占位，不中断构建。
func RenderFence(lang string, content string, ctx Context) (string, bool) {
	for _, f := range fences {
		out, ok, err := f.RenderFence(lang, content, ctx)
		if err != nil {
			log.Printf("[plugins] %s: fence %q error: %v", ctx.Document, lang, err)
			return `<div class="image-layouts-error">Plugin error rendering "` + html.EscapeString(lang) + `": ` + html.EscapeString(err.Error()) + `</div>`, true
		}
		if ok {
			return out, true
		}
	}
	return "", false
}

// CSS 汇总所有插件贡献的 CSS 内容，供构建期合并进 bundle。
func CSS() [][]byte {
	var out [][]byte
	for _, a := range assets {
		out = append(out, a.CSS()...)
	}
	return out
}

// JS 汇总所有插件贡献的 JS 内容，供构建期合并进 bundle。
func JS() [][]byte {
	var out [][]byte
	for _, a := range assets {
		out = append(out, a.JS()...)
	}
	return out
}

// Names 返回已注册插件名（排序），用于启动诊断日志。
func Names() []string {
	names := make([]string, 0, len(plugins))
	for _, p := range plugins {
		names = append(names, p.Name())
	}
	sort.Strings(names)
	return names
}

// String 输出注册摘要，如 "image-layout, my-plugin"。
func String() string {
	return strings.Join(Names(), ", ")
}
