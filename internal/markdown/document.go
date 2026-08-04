package markdown

import "sync/atomic"

// document 记录当前正在渲染的文档名（相对项目根，如 content/posts/foo.md），
// 供插件诊断日志使用。构建为单线程顺序执行，atomic 保证并发安全。
var document atomic.Value

// SetDocument 在每次 markdown 渲染前设置当前文档名，渲染结束后传空串复位。
func SetDocument(name string) {
	document.Store(name)
}

// Document 返回当前渲染的文档名，未设置时为空串。
func Document() string {
	v, _ := document.Load().(string)
	return v
}
