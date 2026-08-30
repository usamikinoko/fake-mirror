# Rainhush 扩展架构

Rainhush 的扩展在**编译期**注册：跨平台、可测试，并且能和静态站构建一起产出内容哈希资源。它不使用 Go 的 `plugin` 动态加载机制（该机制不支持 Windows，也会带来版本兼容与部署问题）。

## 架构边界

```text
pkg/extension        公开扩展协议、registry 与资源收集
pkg/rainhush         自定义二进制的配置与构建入口
extensions/          当前站点的扩展入口（仅空白导入）
internal/builder     内容读取、模板渲染、资源 bundle、站点输出
internal/markdown    Goldmark 适配和内置 Markdown 渲染器
static/js/lifecycle  前端页面生命周期注册表
```

`internal/` 只放实现细节。第三方扩展只能依赖稳定的 `rainhush/pkg/extension` API。

## 创建一个扩展

1. 新建独立 Go module，例如 `example.com/rainhush-box`。
2. 依赖与主站相同版本的 `rainhush` module。
3. 实现 `extension.Extension`；按需实现 `extension.Fence`、`extension.Assets`。
4. 在插件包的 `init()` 中调用 `extension.Register(&plugin{}, 200)`。
5. 在站点的 `extensions/extensions.go` 空白导入该 module。
6. 在 `_config.yaml` 的 `extensions` 中设置开关；省略时默认开启。

```yaml
extensions:
  better-friends: true
  my-box: false
```

配置项只影响已经编译并注册的扩展。设置为 `false` 会同时停用 Fence 渲染和 CSS/JS 资源；未知名称会被忽略，因此同一份配置可以用于不同扩展集合。

最小示例：

```go
package box

import (
    "html"
    "strings"

    "rainhush/pkg/extension"
)

type plugin struct{}

func init() { extension.Register(&plugin{}, 200) }
func (*plugin) Name() string { return "box" }
func (*plugin) RenderFence(lang, content string, _ extension.Context) (string, bool, error) {
    if !strings.HasPrefix(lang, "box-") {
        return "", false, nil
    }
    return `<div class="box">` + html.EscapeString(strings.TrimSpace(content)) + `</div>`, true, nil
}
```

`priority` 越小越先执行：Rainhush 内置扩展使用 `100`，第三方扩展建议使用 `200`。同优先级保持注册顺序。扩展名必须全局唯一；重复注册会立即 panic，避免构建结果依赖导入顺序。

## 资源与前端生命周期

实现 `Assets` 可把 CSS/JS 加入全站哈希 bundle：

```go
func (*plugin) CSS() [][]byte { return [][]byte{cssAsset} }
func (*plugin) JS() [][]byte  { return [][]byte{jsAsset} }
```

扩展 JS 在 bundle 末尾执行。页面首次加载和 Rainhush 的站内局部导航都会触发页面生命周期：

```js
window.registerPageInit(function () {
  document.querySelectorAll(".box").forEach(function (node) {
    if (node.dataset.ready) return;
    node.dataset.ready = "true";
    // 初始化或绑定事件
  });
});
```

初始化函数必须幂等：站内导航会替换 `main`，但也可能因页面事件再次调用初始化器。全局单例（例如 observer）应在重复调用前主动断开或复用。

## 自定义二进制

标准 `rainhush` CLI 会空白导入当前仓库的 `rainhush/extensions` 包。若插件在独立 module，创建一个自己的命令入口：

```go
package main

import (
    "log"
    _ "example.com/my-site/extensions" // 触发所有 extension.Register
    "rainhush/pkg/rainhush"
)

func main() {
    if err := rainhush.BuildSite(); err != nil {
        log.Fatal(err)
    }
}
```

这也是发布带扩展的站点生成器时推荐的方式。

## 错误处理与安全

- 不匹配的围栏必须返回 `handled=false`，让后续扩展或默认代码高亮接管。
- 用户可控文本、属性和 URL 必须 escape 或严格校验后再进入 HTML。
- `error` 用于环境或不可恢复错误；框架会记录日志并输出 `.extension-error` 占位，而不会中断整个站点构建。
- 扩展应保持构建期确定性，不依赖网络或进程全局可变状态。
