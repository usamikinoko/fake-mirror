# 插件模板

复制本目录到 `internal/plugins/<your-plugin>/` 后使用。

## 文件清单

| 文件 | 作用 |
| --- | --- |
| `plugin.go` | 插件骨架：注册、`Fence` 钩子、`Assets` 钩子，示例语法 ```` ```box-<color> ```` |
| `assets/example.css` | 示例样式（可替换或删除） |
| `assets/example.js` | 示例脚本（可替换或删除） |

## 接入步骤

1. 复制目录并改包名（`package myplugin` → 你的包名）与 `Name()`；
2. 重写 `RenderFence` 实现你的语法，记得对不匹配的 lang 返回 `ok=false`；
3. 在 `internal/builder/builder.go` import 区追加空白导入：
   ```go
   _ "rainhush/internal/plugins/<your-plugin>"
   ```
4. `go build ./... && go test ./...` 验证；`rainhush build` 后检查 `Plugins:` 启动日志。

完整开发说明见 `docs/PLUGINS.md`；内置组件（同样基于 fence 渲染）实现参考 `internal/markdown/components/imagelayout/`。
