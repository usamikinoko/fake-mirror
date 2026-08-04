# Rainhush 插件开发指南

Rainhush 有两套语法扩展机制，职责不同：

- **内置组件（components）**：随二进制发布的官方自定义语法，位于 `internal/markdown/components/<name>/`，构建期直接调用，性能优先（`image-layout` 即此类，见第 7 节）。
- **用户插件（plugins）**：面向你自己的自定义扩展，位于 `internal/plugins/<name>/`，通过空白导入注册（本文档主题）。

两套机制都只在构建期拦截**围栏代码块**（```` ```lang `` ```` 或 `~~~lang`），输出任意 HTML，并可向站点全局 bundle 注入 CSS / JS。内置组件优先于用户插件处理。

## 1. 渲染管线中的位置

```
content/*.md ──► goldmark 解析
                    │
                    └─► FencedCodeBlock ──► components.RenderFence（内置组件）
                    │                            │
                    │                      未接管(ok=false)
                    │                            ▼
                    └─► plugins.RenderFence（用户插件）
                    │                            │
                    │            ┌───────────────┴───────────────┐
                    │       插件接管(ok=true)              无人接管
                    │            │                              │
                    │        输出插件 HTML                    默认渲染器
                    │            │                     （mermaid / 代码高亮）
                    ▼
             静态 HTML + bundle CSS/JS
```

- 钩子只处理**围栏代码块**；普通段落、标题等由 goldmark 处理，插件不可干预。
- 内置组件（`components.RenderFence`）先于用户插件；同层按注册顺序轮询，先到先得。
- 渲染发生在**构建期**，产出静态 HTML，无运行时依赖、SEO 友好。
- `~~~lang` 波浪线围栏与 ```` ```lang ```` 走同一路径。

## 2. 目录与注册

插件源码放在 `internal/plugins/<name>/`，通过两处接线完成注册：

1. 插件包在 `init()` 中调用 `plugins.Register(...)`。
2. 在 `internal/builder/builder.go` 的 import 中**空白导入**插件包：

```go
import (
    _ "rainhush/internal/plugins/my-plugin" // 你的插件
)
```

空白导入触发插件包的 `init()`，把插件注册进全局注册表。启用/停用一个插件 = 增删这一行 import。

## 3. 核心接口

全部定义在 `internal/plugins/plugins.go`：

```go
// Plugin 是所有插件的公共接口，注册的插件至少需要提供唯一名称。
type Plugin interface {
    Name() string
}

// Fence 是可选的围栏代码块渲染钩子。
// lang 是围栏语言（如 image-layout-d、mermaid、custom-xyz），content 是块内原始文本。
// 返回 ok=true 表示该插件接管渲染（输出 html）；false 表示不处理。
type Fence interface {
    RenderFence(lang string, content string, ctx Context) (html string, ok bool, err error)
}

// Assets 是可选的资源提供钩子，返回需要合并进站点全局 bundle 的 CSS / JS 内容。
type Assets interface {
    CSS() [][]byte
    JS() [][]byte
}

// Context 是渲染钩子的调用上下文，携带当前渲染的文档名供诊断日志使用。
type Context struct {
    Document string
}
```

要点：

- 只实现 `Plugin` 的插件可注册但不做事（如仅提供全局 CSS 的主题插件）。
- `Fence` / `Assets` 是**可选接口**，通过类型断言探测，未实现不影响注册。
- `content` 是块内原始文本（不含围栏标记），首行保留前导换行，插件自行 `strings.Trim*`。
- `err` 非 nil 时，框架会记录日志并输出 `<div class="image-layouts-error">` 错误占位，**不中断构建**——这是有意设计：单个语法错误不应让整个站点构建失败。用户可读的校验错误（如非法网格）应直接作为 HTML 输出，不要返回 err。

## 4. 开发一个插件（分步）

以把 ```` ```box-red ```` 渲染为彩色边框容器的插件为例：

**第 1 步：建目录与骨架**

`internal/plugins/my-box/plugin.go`：

```go
package mybox

import (
    "html"
    "log"
    "strings"

    "rainhush/internal/plugins"
)

type plugin struct{}

var _ plugins.Fence = (*plugin)(nil)

func init() { plugins.Register(&plugin{}) }

func (p *plugin) Name() string { return "my-box" }
```

**第 2 步：实现 Fence 钩子**

```go
const prefix = "box-"

func (p *plugin) RenderFence(lang string, content string, ctx plugins.Context) (string, bool, error) {
    if !strings.HasPrefix(lang, prefix) {
        return "", false, nil // 不处理，交给其他插件或默认渲染器
    }
    color := strings.TrimPrefix(lang, prefix)
    log.Printf("[%s] %s: box color=%q content=%d chars", p.Name(), ctx.Document, color, len(content))
    return `<div class="my-box my-box-` + html.EscapeString(color) + `">` +
        html.EscapeString(strings.TrimSpace(content)) + `</div>`, true, nil
}
```

注意：所有用户可控内容（颜色名、正文）必须经 `html.EscapeString` 再拼入输出，防止注入。

**第 3 步：接线注册**

在 `internal/builder/builder.go` import 区追加：

```go
_ "rainhush/internal/plugins/my-box"
```

构建后 ```` ```box-red hello ```` 即渲染为 `<div class="my-box my-box-red">hello</div>`，而 ```` ```go ```` 等普通代码块不受影响（插件返回 `false`，落入默认高亮渲染器）。

**第 4 步：提供样式与脚本（可选）**

用 `embed` 把静态资源编译进二进制，随 bundle 自动发布：

```go
import _ "embed"

//go:embed assets/my-box.css
var cssAsset []byte

//go:embed assets/my-box.js
var jsAsset []byte

func (p *plugin) CSS() [][]byte { return [][]byte{cssAsset} }
func (p *plugin) JS() [][]byte  { return [][]byte{jsAsset} }
```

插件 CSS 追加在站点内置 CSS 之后、JS 追加在内置 JS 之后，合并进带内容哈希的 `bundle.<hash>.css/js`（构建期自动完成，无需其他配置；模板与缓存策略已覆盖）。

**第 5 步：写测试**

每个插件应有单元测试覆盖解析与渲染逻辑，参照 `internal/markdown/components/imagelayout/imagelayout_test.go`（内置组件的测试，同样是 fence 渲染测试范式）。测试中直接调用你的 `RenderFence` 即可，无需走完整构建。

## 5. 日志与调试

- 插件日志统一使用标准库 `log.Printf`，前缀 `[<插件名>]`，例如 `[image-layout]`。
- `ctx.Document` 携带当前渲染文档的路径（如 `content/posts/foo.md`），块级日志务必带上，便于定位。
- 构建启动时会打印 `Plugins: <已注册插件名列表>`，先确认你的插件出现在列表里。
- 日志输出到 stderr，`rainhush build` 时直接可见；`rainhush test` 模式下每次热重建都会输出。

## 6. 性能与规范

- **正则一次性编译**：包级 `var re = regexp.MustCompile(...)`，不要在渲染路径里反复编译。
- **零分配优先**：用 `strings.Builder` 拼 HTML；`render` 路径避免反射与 map 重解析。
- **短路返回**：不匹配的 fence 尽快返回 `( "", false, nil )`，不执行任何解析。
- **构建期纯函数**：钩子内不得依赖运行时状态；`Context` 之外不要引入全局可变状态。
- **错误分级**：用户输入校验失败 → 输出 `.image-layouts-error` 样式提示（插件自己的 HTML）；环境/IO 异常 → 返回 err 交给框架兜底。
- **命名规范**：包名用目录名（下划线转驼峰），`Name()` 返回的插件名全局唯一且保持稳定。

## 7. 内置组件与插件系统

**内置组件（components）** 是随二进制发布的官方自定义语法，位于 `internal/markdown/components/`：

- 注册中心 `internal/markdown/components/components.go` 定义组件接口并分发：`Render(lang, content, doc) (html, ok)` 处理围栏、`CSS()`/`JS()` 注入 bundle。
- 每个组件一个子目录（如 `imagelayout/`），在 `init()` 中 `components.Register(...)`；`internal/markdown/codeblock.go` 空白导入组件包触发注册。
- 新增内置组件 = 新建子目录 + 实现接口 + `codeblock.go` 加一行空白导入。`image-layout`（图片结构化布局，语法见 `docs/MIGRATION-GUIDE.md`）即内置组件示例。

**用户插件（plugins）** 面向你的自定义扩展，见第 2~6 节。两者互不依赖，内置组件始终优先。

## 8. 标准模板

`docs/plugin-template/` 提供可直接复用的插件骨架（含 Fence 示例、Assets 示例、CSS/JS 占位文件与使用说明）。复制该目录到 `internal/plugins/<你的插件名>/`，改包名、`Name()`、渲染逻辑，接好 import 即可。

常见问题：

- **插件没生效** → 检查 builder.go 的空白 import 是否添加；`Plugins:` 启动日志是否包含插件名；fence 语言前缀是否匹配。
- **普通代码块被吞** → 你的 `RenderFence` 必须对不匹配的 lang 返回 `ok=false`。
- **样式没出现** → 确认实现了 `Assets` 接口且 embed 路径正确；CSS 选择器避免与站点内置样式重名。
