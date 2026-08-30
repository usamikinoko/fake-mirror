# Rainhush 扩展开发指南

Rainhush 使用**编译期扩展**：插件作为 Go module 与站点生成器一起编译，通过公开的 `rainhush/pkg/extension` 注册。该方案跨平台、可测试；不使用 Go `plugin` 动态加载（Windows 不支持，且会引入 ABI/版本风险）。

架构总览与自定义二进制示例请先阅读 [`EXTENSIONS.md`](./EXTENSIONS.md)。

## 1. 接入位置

标准 Rainhush CLI 在启动时导入仓库根目录的 `extensions` 包：

```go
import _ "rainhush/extensions"
```

在 `extensions/extensions.go` 中添加第三方插件的空白导入：

```go
package extensions

import _ "example.com/rainhush-my-box"
```

插件会在 `init()` 中向全局 registry 注册。修改扩展后重新编译二进制即可；构建期会输出 `Extensions: ...` 作为诊断信息。

## 2. 配置开关

扩展是否参与当前站点构建由 `_config.yaml` 控制：

```yaml
extensions:
  better-friends: true
  image-layout: false
```

扩展已编译但配置中没有对应项时默认开启。只有显式设置为 `false` 才会关闭。关闭后该扩展的 Fence 不会接管内容，其 CSS/JS 也不会写入 bundle。配置只能关闭已注册扩展，不能动态加载尚未编译进二进制的扩展；新增扩展仍需空白导入并重新编译。

## 2. 公开接口

```go
package extension

type Context struct { Document string }

type Extension interface { Name() string }

type Fence interface {
    RenderFence(lang string, content string, ctx Context) (output string, handled bool, err error)
}

type Assets interface {
    CSS() [][]byte
    JS() [][]byte
}

func Register(ext Extension, priority int)
```

- 实现 `Extension` 是必需的；`Fence` 和 `Assets` 是可选能力。
- `handled=false` 表示该扩展不处理此围栏，后续扩展或默认代码高亮继续处理。
- `priority` 越小越优先。内置扩展为 `100`，第三方推荐 `200`。
- 同优先级保持注册顺序；名称重复会立即 panic，避免不确定的构建结果。

## 3. 示例

```go
package mybox

import (
    _ "embed"
    "html"
    "strings"

    "rainhush/pkg/extension"
)

type plugin struct{}

func init() { extension.Register(&plugin{}, 200) }
func (*plugin) Name() string { return "my-box" }

func (*plugin) RenderFence(lang, content string, ctx extension.Context) (string, bool, error) {
    if !strings.HasPrefix(lang, "box-") {
        return "", false, nil
    }
    return `<div class="my-box">` + html.EscapeString(strings.TrimSpace(content)) + `</div>`, true, nil
}
```

用法：

````markdown
```box-red
Hello
```
````

完整可复制模板位于 [`plugin-template/`](./plugin-template/)。

## 4. 前端资源

实现 `Assets` 可把 CSS/JS 注入站点的内容哈希 bundle：

```go
//go:embed assets/box.css
var cssAsset []byte

func (*plugin) CSS() [][]byte { return [][]byte{cssAsset} }
```

页面可通过局部导航切换，插件 JS 不应只依赖 `DOMContentLoaded`。请注册幂等初始化器：

```js
window.registerPageInit(function () {
  document.querySelectorAll(".my-box").forEach(function (el) {
    if (el.dataset.ready) return;
    el.dataset.ready = "true";
  });
});
```

初始化器在首屏和每次 `pagechange` 后都会执行。

## 5. 安全与质量

- 对围栏内容、属性和 URL 使用 `html.EscapeString` 或严格校验。
- 不要以网络请求或全局可变状态决定构建结果。
- 用户输入错误应输出可读的插件提示；环境/I/O 错误可返回 `error`，框架会记录并输出 `.extension-error`，不中断整站构建。
- 为匹配、未匹配和异常路径写单元测试。registry 本身已覆盖优先级、资源排序和重复名称检测。

> 旧 `internal/plugins` API 目前仍可用，但只作为兼容 facade。新插件请始终使用 `pkg/extension`。
