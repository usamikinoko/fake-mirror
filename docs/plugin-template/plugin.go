// 插件标准模板。
// 用法：复制本目录到 internal/plugins/<your-plugin>/，然后：
//  1. 包名改为你的插件名（目录名，下划线转驼峰）；
//  2. 修改 Name() 返回唯一插件名；
//  3. 重写 RenderFence 实现你的语法（参考 internal/markdown/components/imagelayout/ 的渲染范式）；
//  4. 按需替换 assets/ 下的 CSS/JS 文件；
//  5. 在 internal/builder/builder.go 的 import 中追加空白导入：
//     _ "rainhush/internal/plugins/<your-plugin>"
//  6. 运行 go build ./... 与 go test ./... 验证。
//
// 完整说明见 docs/PLUGINS.md。
package myplugin

import (
	_ "embed"
	"html"
	"log"
	"strings"

	"rainhush/internal/plugins"
)

//go:embed assets/example.css
var cssAsset []byte

//go:embed assets/example.js
var jsAsset []byte

type plugin struct{}

var _ plugins.Fence = (*plugin)(nil)
var _ plugins.Assets = (*plugin)(nil)

func init() { plugins.Register(&plugin{}) }

func (p *plugin) Name() string { return "my-plugin" }

func (p *plugin) CSS() [][]byte { return [][]byte{cssAsset} }
func (p *plugin) JS() [][]byte  { return [][]byte{jsAsset} }

// fencePrefix 是本插件接管的围栏语言前缀，例如 ```box-red 匹配 "box-"。
const fencePrefix = "box-"

// RenderFence 示例：把 ```box-<color> <content> 渲染为彩色边框容器。
// 不匹配的 lang 必须返回 ok=false，避免吞掉普通代码块。
func (p *plugin) RenderFence(lang string, content string, ctx plugins.Context) (string, bool, error) {
	if !strings.HasPrefix(lang, fencePrefix) {
		return "", false, nil
	}
	color := strings.TrimPrefix(lang, fencePrefix)
	log.Printf("[%s] %s: box color=%q content=%d chars", p.Name(), ctx.Document, color, len(content))
	var b strings.Builder
	b.WriteString(`<div class="my-box my-box-`)
	b.WriteString(html.EscapeString(color))
	b.WriteString(`">`)
	b.WriteString(html.EscapeString(strings.TrimSpace(content)))
	b.WriteString(`</div>`)
	return b.String(), true, nil
}
