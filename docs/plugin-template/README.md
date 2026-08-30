# Rainhush 扩展模板

把本目录复制到独立 Go module 后使用。新扩展只依赖公开的 `rainhush/pkg/extension`，不应导入 `internal/`。

## 文件清单

| 文件 | 作用 |
| --- | --- |
| `plugin.go` | 扩展骨架：注册、`Fence` 钩子、`Assets` 钩子，示例语法 ```` ```box-<color> ```` |
| `assets/example.css` | 示例样式（可替换或删除） |
| `assets/example.js` | 示例脚本（可替换或删除） |

## 接入步骤

1. 复制目录，改包名（`package myplugin`）与唯一的 `Name()`；
2. 重写 `RenderFence`，不匹配时返回 `handled=false`；
3. 在站点项目的 `extensions/extensions.go` 添加空白导入：
   ```go
   import _ "example.com/your-extension"
   ```
4. 编译自定义 Rainhush 二进制；构建日志应出现 `Extensions: <name>`。
5. 用 fenced Markdown、资源 bundle 和站内局部导航分别验证你的渲染/前端行为。

完整 API、优先级、资源与页面生命周期约定见 [`../EXTENSIONS.md`](../EXTENSIONS.md)。内置实现可参考 `internal/markdown/components/imagelayout/`，但第三方扩展不应依赖其内部包。
