# Rainhush

Rainhush 是一个基于 Go 语言的轻量级静态站点生成工具 (Static Site Generator)。它将 Markdown 内容转换为完整渲染的静态站点，支持语法高亮、Mermaid 图表渲染，并内置部署工具。

### 语言

[English](./README.md) | [中文](./README_CN.md)

开发流程手册见 [DEVELOPMENT.md](./DEVELOPMENT.md)

## 快速入门

#### 安装

```bash
# npm (推荐)
npm install -g rainhush

# 或：从源码编译
go install github.com/usamikinoko/rainhush@latest
```

#### 创建站点

```bash
cp _config.example.yaml _config.yaml
rainhush build   # 构建站点到 public/
rainhush test    # 构建、本地预览并监听文件变化自动重建
rainhush push    # 构建并部署
```

## CLI 命令

| 命令 | 描述 |
|---------|------|
| `rainhush build` | 将站点构建到 `public/` |
| `rainhush test` | 构建，启动本地服务器，并在 `content/`, `templates/`, `static/` 中的文件变化时自动重建 |
| `rainhush push` | 构建并部署生成的站点 |
| `rainhush clean` | 删除 `public/` 目录 |
| `rainhush clear` | `rainhush clean` 的兼容别名 |
| `rainhush --version` | 打印版本号 |

注意：

- `test` 会监听文件变化并自动重建，但不会注入浏览器热更新。
- `build` 会保留 `public/.git`，所以基于 Git 的部署方案可以在多次构建之间保持远程配置和历史记录。

## 配置

将 `_config.example.yaml` 复制为 `_config.yaml`：

```yaml
server:
  port: 8080

site:
  url: https://example.com
  description: 你的站点 SEO 描述
  favicon: https://example.com/favicon.jpg

home:
  title: Rainhush
  subtitle: My Blog
  avatar: https://example.com/avatar.jpg
  owner: Rainhush User

deploy:
  mode: git
  remote: git@github.com:username/repo.git
  branch: gh-pages
```

字段说明：

- `home.title` 作为站点标题用于页眉、页脚、页面 `<title>` 和 RSS 订阅源。
- `deploy.remote` 在 `git` 模式下为必填项，可以是：
  - 一个 Git 仓库 URL，如 `git@github.com:username/repo.git`
  - 一个已在 `public/.git` 中配置好的远程仓库名称

## 部署模式

#### Git 模式

`deploy.mode: git` 将构建后的 `public/` 目录推送到 Git 仓库。

- 如果 `deploy.remote` 是 Git URL，Rainhush 会自动配置一个 `deploy` 远程仓库。
- 如果 `deploy.remote` 是远程仓库名称，则该名称必须已在 `public/.git` 中存在。
- 推荐将部署分支设置为 `gh-pages`；`push` 会对生成站点分支执行强推送。

#### Server 模式

`deploy.mode: server` 通过 SSH/SFTP 将 `public/` 上传到 Linux 服务器，并原子化切换发布版本。

支持的服务器配置：

```yaml
deploy:
  mode: server
  server:
    host: example.com
    port: 22
    user: deploy
    path: /var/www/rainhush
    identity: C:/Users/you/.ssh/id_ed25519
    known_hosts: C:/Users/you/.ssh/known_hosts
    # 当密钥认证无法使用时的可选备选
    password: your-password
```

## 内容结构

```text
content/
├── posts/
├── about/
│   └── about.md
└── friends/
    └── friends.md
```

#### 文章

文章放在 `content/posts/` 目录中，采用 YAML 前置元数据加 Markdown 内容的格式。

```markdown
---
title: My First Post
author: Your Name
date: 2026-07-01
updated_at: 2026-07-05
location: Shanghai
avatar: https://example.com/avatar.jpg
cover: https://example.com/cover.jpg
category: technology
---

Write your post content here.

<!-- more -->

More content here.
```

支持的字段：

| 字段 | 必填 | 描述 |
|-------|------|------|
| `title` | 是 | 文章标题 |
| `date` | 是 | 发布日期，格式为 `YYYY-MM-DD` |
| `author` | 否 | 作者名称 |
| `updated_at` | 否 | 更新日期，格式为 `YYYY-MM-DD` |
| `location` | 否 | 地理位置 |
| `avatar` | 否 | 作者头像 URL |
| `cover` | 否 | 封面图片 URL |
| `category` | 否 | `technology` 或 `life`；影响首页分栏展示 |

实现细节：

- `title` 和 `date` 会在构建时进行校验。
- Mermaid 支持仅在存在真实的代码块 ```mermaid``` 时启用。
- `<!-- more -->` 定义了首页和订阅源摘要的截断点。

#### About 和 Friends

`content/about/about.md` 和 `content/friends/friends.md` 会作为普通 Markdown 页面渲染，支持可选的 `title` 前置元数据。

当前行为：

- Friends 链接没有特殊的卡片模式。
- 如果未填写 `title`，页面会默认显示为 `About` 或 `Friends`。
