// Package components 是内置代码块组件（自定义围栏语法）的注册与分发中心。
// 每个内置组件（如 imagelayout）负责一种或一类围栏语法，随二进制发布、构建期直接调用；
// 用户自定义扩展请使用 internal/plugins 插件系统。
package components

import (
	"log"
	"sort"
	"strings"
)

// Component 是内置代码块组件接口：
// Render 处理围栏代码块，lang 为围栏语言、content 为块内原始文本、doc 为当前渲染文档路径（日志用）；
// 返回 ok=true 表示已接管渲染。CSS/JS 返回需要合并进站点全局 bundle 的资产。
type Component interface {
	Name() string
	Render(lang string, content string, doc string) (html string, ok bool)
	CSS() [][]byte
	JS() [][]byte
}

var registry []Component

// Register 注册一个内置组件，在组件包的 init() 中调用。
func Register(c Component) {
	registry = append(registry, c)
}

// RenderFence 按注册顺序把围栏代码块分发给各内置组件，首个接管（ok=true）的组件决定输出。
func RenderFence(lang string, content string, doc string) (string, bool) {
	for _, c := range registry {
		out, ok := c.Render(lang, content, doc)
		if !ok {
			continue
		}
		if out == "" {
			log.Printf("[components] %s: component %q returned empty html for lang %q", doc, c.Name(), lang)
		}
		return out, true
	}
	return "", false
}

// CSS 汇总所有组件贡献的 CSS 内容，供构建期合并进 bundle。
func CSS() [][]byte {
	var out [][]byte
	for _, c := range registry {
		out = append(out, c.CSS()...)
	}
	return out
}

// JS 汇总所有组件贡献的 JS 内容，供构建期合并进 bundle。
func JS() [][]byte {
	var out [][]byte
	for _, c := range registry {
		out = append(out, c.JS()...)
	}
	return out
}

// Names 返回已注册组件名（排序），用于启动诊断日志。
func Names() []string {
	names := make([]string, 0, len(registry))
	for _, c := range registry {
		names = append(names, c.Name())
	}
	sort.Strings(names)
	return names
}

// String 输出组件注册摘要，如 "image-layout"。
func String() string {
	return strings.Join(Names(), ", ")
}
