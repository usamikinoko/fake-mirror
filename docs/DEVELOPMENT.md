# 开发手册

## 本地开发命令

```powershell
go run . build
go run . test
go run . clean
go run . push
go test ./...
npm test
```

说明：

- `go run . build`：构建静态站点到 `public/`
- `go run . test`：先构建，再启动本地预览，并监听 `content/`、`templates/`、`static/`
- `go run . clean`：清理 `public/`
- `go run . push`：先构建，再按 `_config.yaml` 中的部署配置推送
- `go test ./...` / `npm test`：运行 Go 测试
- Git 部署模式建议使用 `deploy.branch: gh-pages`，因为 `push` 会覆盖部署分支内容

## 推送日常更新

```powershell
git status
go test ./...
go run . build
git add .
git commit -m "feat: describe your change"
git push origin <branch>
```

建议顺序：

1. 先跑 `go test ./...`
2. 如果改动影响站点渲染，再跑 `go run . build`
3. 确认 `public/`、`rainhush.exe`、`.tmp/` 等产物没有被加入提交
4. 再执行 `git add` / `git commit` / `git push`

## 发布新的 GitHub Release

当前仓库已经配置了 GitHub Actions：当你推送形如 `v0.1.8` 的 tag 时，会自动：

1. 运行 `go test ./...`
2. 同步 `package.json` 版本号
3. 构建带版本号的 `rainhush.exe`
4. 发布 npm 包
5. 创建同名 GitHub Release，并附带 `rainhush.exe`

发布步骤：

```powershell
git checkout main
git pull --ff-only origin main
go test ./...
git tag v0.1.8
git push origin main
git push origin v0.1.8
```

发布前请同步更新 `CHANGELOG.md`，并保证其中存在对应版本的 `## v0.1.8` 小节，否则 Release 说明会为空。

## 基于新 Release 发布新的 npm 版本

这里不需要额外手工执行 `npm publish`。npm 发版已经绑定在 tag 发布流程里：

1. 先更新源码和 `CHANGELOG.md`
2. 推送新的语义化 tag，例如 `v0.1.8`
3. 等待 GitHub Actions 的 `Release & Publish` 工作流完成
4. 到 npm 页面确认新版本已发布

如果你只想在本地验证 npm 包内容，可以运行：

```powershell
npm run prepack
npm pack --dry-run
```

这会先在当前目录构建新的 `rainhush.exe`，再检查最终会被打进 npm 包的文件列表。
