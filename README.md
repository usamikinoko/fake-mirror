# Rainhush

Rainhush is a fast, lightweight static site generator built with Go. It transforms Markdown content into a fully-rendered static site with syntax highlighting, Mermaid diagram support, structured image layouts, and built-in deployment tooling.

### Language

[English](./README.md) | [中文](./README_CN.md)

Developer workflow: [DEVELOPMENT.md](./DEVELOPMENT.md)

## Features

- **Syntax highlighting** — Chroma-powered code highlighting with copy buttons.
- **Mermaid diagrams** — Render `mermaid` fenced code blocks at build time.
- **Structured image layouts** — Display images as curated grids, masonry columns, carousels, or custom layouts with the built-in `image-layout` syntax (details below).

### Image Layouts

Rainhush embeds the `image-layout` syntax for turning fenced code blocks into structured image displays. It is compatible with [obsidian-image-layouts](https://github.com/git-no/obsidian-image-layouts), so Obsidian notes can be published as-is without syntax changes. Rendering happens at build time — output is plain static HTML, SEO-friendly and dependency-free.

```markdown
```image-layout-a
![[beach-1.jpg|Low tide]]
![[beach-2.jpg|Sunset]]
```

```image-layout-masonry-3
![[photo-1.jpg]]
![[photo-2.jpg]]
![[photo-3.jpg]]
![[photo-4.jpg]]
![[photo-5.jpg]]
![[photo-6.jpg]]
```
```

Options can be set via block-level YAML front matter:

```markdown
```image-layout
---
layout: d
caption: A day at the beach
overlay: always
descriptions:
  - Low tide
  - Running in the sand
  - Sunset
---
![[beach-1.jpg]]
![[beach-2.jpg]]
![[beach-3.jpg]]
```
```

Notes:

- `![[name.jpg]]` resolves to `static/images/name.jpg`, served at `/images/...`; remote URLs (`https://...`) and Markdown image syntax `![alt](url)` are also supported.
- Available layouts: preset grids `a`–`i` and `single`, masonry `masonry-2`…`masonry-6`, carousel `carousel`, custom ASCII grids (`grid:` option), plus `image-layout-left/center/right` alignment shortcuts.
- Performance: image dimensions are detected at build time and emitted as `width`/`height` (no layout shift); the first image loads eagerly with high priority while the rest lazy-load on scroll; grid cells use a fixed row height.
- Full syntax specification: [MIGRATION-GUIDE.md](./docs/MIGRATION-GUIDE.md)

## Quick Start

#### Install

```bash
# npm (recommended)
npm install -g rainhush

# or: build from source
go install github.com/usamikinoko/rainhush@latest
```

#### Create a site

```bash
cp _config.example.yaml _config.yaml
rainhush build   # Build the site into public/
rainhush test    # Build, serve locally, and rebuild on file changes
rainhush push    # Build and deploy
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `rainhush build` | Build the site into `public/` |
| `rainhush test` | Build, serve locally, and rebuild when files under `content/`, `templates/`, or `static/` change |
| `rainhush push` | Build and deploy the generated site |
| `rainhush clean` | Remove `public/` |
| `rainhush clear` | Alias of `rainhush clean` |
| `rainhush --version` | Print version |

Notes:

- `Test` does file watching and rebuilds, but it does **not** inject browser live reload.
- `Build` preserves `public/.git`, so Git-based deployments keep their remote configuration and history between builds.

## Configuration

Copy `_config.example.yaml` to `_config.yaml`:

```yaml
server:
  port: 8080

site:
  url: https://example.com
  description: Your site description for SEO
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

Field notes:

- `home.title` is used as the site title in the header, footer, page `<title>`, and RSS feed.
- `deploy.remote` is required in `git` mode. It can be either:
  - a repository URL such as `git@github.com:username/repo.git`
  - an existing remote name already configured inside `public/.git`

## Deploy Modes

#### Git mode

`deploy.mode: git` pushes the built `public/` directory to Git.

- If `deploy.remote` is a Git URL, Rainhush configures an internal `deploy` remote automatically.
- If `deploy.remote` is a remote name, that remote must already exist in `public/.git`.
- The recommended deploy branch is `gh-pages`; `push` force-pushes the generated site branch.

#### Server mode

`deploy.mode: server` uploads `public/` to a Linux server over SSH/SFTP and swaps releases atomically.

Supported server fields:

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
    # Optional fallback when key auth is unavailable
    password: your-password
```

## Content Structure

```text
content/
├── posts/
├── about/
│   └── about.md
└── friends/
    └── friends.md
```

#### Posts

Posts live in `content/posts/` and use YAML frontmatter plus Markdown content.

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

Supported fields:

| Field | Required | Description |
|-------|--------|------------|
| `title` | Yes | Post title |
| `date` | Yes | Publication date in `YYYY-MM-DD` format |
| `author` | No | Author name |
| `updated_at` | No | Updated date in `YYYY-MM-DD` format |
| `location` | No | Geographic location |
| `avatar` | No | Author avatar URL |
| `cover` | No | Cover image URL |
| `category` | No | `technology` or `life`; affects the homepage column |

Implementation details:

- `title` and `date` are validated at build time.
- Mermaid support is enabled only when a real fenced `mermaid` code block exists.
- `<!-- more -->` defines the excerpt boundary for homepage and feed summaries.

### About and Friends

`content/about/about.md` and `content/friends/friends.md` are rendered as plain Markdown pages with optional `title` frontmatter.

Current behavior:

- There is no special card schema for friends links.
- If `title` is omitted, the page falls back to `About` or `Friends`.
