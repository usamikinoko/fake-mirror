# Obsidian Image Layouts → 静态博客迁移指南

> **文档性质**：本文档是 `obsidian-image-layouts` 插件（v0.18.0）的**权威语法规范**与**静态站点迁移实施依据**。
> 阅读对象：目标项目（Hugo 或类 Hugo 静态博客）的开发者 / Agent。
> 使用方法：将此文件移入目标项目仓库，由 Agent 严格按「第 6~10 章」实施，并按「第 11~12 章」验收。
>
> 源码依据：本仓库 `src/`（插件运行时）、`test-vault/publish.ts`（插件作者维护的 Obsidian Publish 静态渲染参考实现）。

---

## 目录

1. [背景与目标](#1-背景与目标)
2. [总体结论（你的方案是否合适）](#2-总体结论)
3. [语法规范（权威参考）](#3-语法规范)
4. [渲染行为规范（必须匹配）](#4-渲染行为规范)
5. [Obsidian 依赖剥离清单与源码映射](#5-obsidian-依赖剥离清单与源码映射)
6. [迁移方案论证与选型](#6-迁移方案论证与选型)
7. [方案 A：Hugo Code Block Render Hooks（标准 Hugo ≥ 0.127，推荐）](#7-方案-ahugo-code-block-render-hooks)
8. [方案 B：Node/TS 构建期预处理器（自研 / 非 Hugo 项目，通用推荐）](#8-方案-bnode-预处理器)
9. [共享资产：CSS 与 Carousel JS](#9-共享资产css-与-carousel-js)
10. [实施步骤清单](#10-实施步骤清单)
11. [验收测试清单](#11-验收测试清单)
12. [附录：插件源码文件 → 迁移策略映射](#12-附录插件源码文件--迁移策略映射)

---

## 1. 背景与目标

`obsidian-image-layouts` 是 Obsidian 插件，为笔记提供**图片代码块**语法 `image-layout-*`，把一个代码块渲染成一组美观的图片布局（网格、瀑布流、轮播、自定义网格）。

**迁移目标**：把该插件的**语法**与**图片展示能力**搬到目标静态博客站点生成项目（如 Hugo），使得：

- Obsidian 仓库中的既有笔记**语法零改动**即可发布到博客（这是最高优先级，见 §6）；
- 渲染发生在**构建期**（静态 HTML，无 JS 依赖、SEO 友好）；
- 布局效果与原插件一致（网格比例、间距、overlay、caption 等）。

---

## 2. 总体结论

你提出的「先读源码 → 整理成 MD 文档 → 移入目标项目 → 让 Agent 按文档开发」的流程**总体合理**，是本类迁移的最佳实践之一：

| 优点 | 说明 |
| --- | --- |
| 跨项目载体 | 文档可随仓库版本化、可审计、可复用，不绑定 Obsidian 环境 |
| 阻断 API 耦合 | 插件代码大量依赖 Obsidian API（vault、metadataCache、渲染上下文），文档把它们显式隔离，防止 Agent 盲目搬运 |
| 可验收 | 文档包含验收清单，Agent 交付后可按清单核对 |

但有 **3 点必须修正**，否则 Agent 会走弯路：

1. **文档不是「代码整理」，而是「规范 + 实现指引」**。插件源码中可复用的只有纯逻辑（正则解析、grid 解析、YAML 解析），渲染层（Svelte 组件、CSS-in-JS、UnoCSS）与 Obsidian API 强绑定，无法直接搬运，必须重写为静态 HTML/CSS。
2. **必须显式列出「Obsidian 专属、需剥离」的部分**：布局选择器（LayoutPicker）、布局切换按钮（SwitchableLayout）、设置面板、编辑回写（editor-writeback）。这些是编辑态交互，静态站点完全不需要（见 §5）。
3. **文档必须附可直接运行的模板代码**（Hugo 模板 / 预处理脚本 / CSS），而不是只描述行为，减少 Agent 的推断空间。

本指南已按上述原则编写。**推荐的迁移路线**（详见 §6）：

> **标准 Hugo（≥ 0.127）→ 方案 A：Code Block Render Hooks（语法零改动、构建期渲染）**
> **自研或其它 SSG → 方案 B：Node/TS 构建期预处理器（复用插件纯逻辑，保真度最高）**
> **两者皆可叠加方案 D：客户端 JS（仅用于 Carousel 等交互组件）**

---

## 3. 语法规范

### 3.1 代码块（fence）变体

语法载体是 Markdown 围栏代码块，fence 使用 `` ``` `` 或 `~~~`（长度 ≥ 3）。有两代语法，**两者都必须支持**：

#### 3.1.1 Legacy 语法（fence 名带布局后缀）

| Fence 语言 | 布局 |
| --- | --- |
| `image-layout-a` … `image-layout-i` | 预置网格（见 §3.4） |
| `image-layout-single` | 单图网格 |
| `image-layout-left` / `center` / `right` | 单图 + 水平对齐（等价 `single` + `align`） |
| `image-layout-masonry-2` … `image-layout-masonry-6` | 瀑布流，2~6 列 |

#### 3.1.2 Modern 语法（`image-layout` + front matter）

````
```image-layout
---
layout: a            # a~i / single / masonry-2~6 / carousel / custom
caption: 标题
---
![[beach-1.jpg|Low tide]]
![[beach-2.jpg]]
```
````

`layout` 字段取值（兼容 `legacy-layout-*` / `legacy-masonry-*` 前缀写法，如 `legacy-layout-a` 等价 `a`、`legacy-masonry-3` 等价 `masonry-3`）：

| 值 | 说明 |
| --- | --- |
| `a` ~ `i`、`single` | 预置网格 |
| `masonry-2` ~ `masonry-6` | 瀑布流 |
| `carousel` | 轮播 |
| `custom` | 自定义 ASCII 网格（需 `grid` 选项） |

> 插件中**空的** `image-layout` 块会显示交互式布局选择器 —— 这是编辑态 UI，**静态站点中不渲染任何内容**（可输出注释占位），但不应报错。

### 3.2 图片行语法

代码块**正文**中，每行一个图片，**只有以 `!` 开头的行**才会被收集。支持两种格式：

| 格式 | 示例 | 说明 |
| --- | --- | --- |
| Wiki 链接 | `![[beach.jpg]]` | 本地图片（相对仓库根） |
| Wiki 链接 + 描述 | `![[beach.jpg|Low tide]]` | `|` 后非数字段为 overlay 文本 |
| Wiki 链接 + 尺寸 | `![[beach.jpg|300]]`、`![[beach.jpg|300x200]]` | 纯数字 → 宽（px）；`WxH` → 宽高 |
| Wiki 链接 + 混合 | `![[beach.jpg|Low tide|300]]` | 数字段为尺寸，其余段拼为描述（多段以 `|` 连接） |
| Markdown | `![Low tide](beach.jpg)` | alt 文本同样按 `|` 分段解析 |
| Markdown 远程 | `![Low tide](https://example.com/x.jpg)` | `http(s)://` 视为远程 |
| Markdown 本地 | `![](file:///abs/path/x.jpg)` | 绝对路径（迁移后可忽略此形态） |

**解析规则（重要）**：

- `|` 分段：`trim` 后是**纯数字** `\d+` → `width`；`\d+x\d+` → `width`+`height`；**其他非空段** → 描述文本（多段用 `|` 拼接）。
- Markdown 链接目标允许一层嵌套括号（如 `Screenshot (1).png`）。
- 描述文本（`alt`）在渲染时作为图片 overlay / alt 属性的来源。
- 图片路径若含 Obsidian 百分号编码（如 `Test%20folder/img.jpg`），解析时先做一次 `decodeURIComponent`，失败则用原串。

### 3.3 Front Matter 选项（全部）

代码块内部可携带 YAML front matter（以 `---` 包裹），下列选项适用所有布局：

| 选项 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `caption` | 字符串 | 无 | 整个布局下方的说明文字（居中、小字、弱化色） |
| `descriptions` | 字符串数组 | 无 | 逐图 overlay 文本，**优先级高于**图片行自带描述（`descriptions[i]` > 图片行 alt） |
| `overlay` | `never` / `hover` / `always` | `hover` | 逐图 overlay 的显示时机 |
| `permanentOverlay` | 布尔 | 无 | 旧写法：`true` ⇔ `overlay: always`，`false` ⇔ `hover`；**低于** `overlay` 字段 |
| `fromFolder` | 字符串 | 无 | 额外引入仓库某文件夹下的全部图片（**仅直接子文件**），排在块内图片之后 |
| `sortBy` | `name` / `mtime` | `name` | `fromFolder` 图片排序 |
| `reverse` | 布尔 | `false` | 反转 `fromFolder` 排序 |
| `limit` | 数字 | 无 | 截断 `fromFolder` 图片数量 |

网格布局（`a`~`i`、`single`）专属：

| 选项 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `fit` | `cover` / `contain` / `natural` | `cover` | 图片填充网格单元格的方式（见 §4.4） |
| `align` | `left` / `center` / `right` / `full` | `full` | 整个布局的水平放置；非 `full` 时容器宽度默认 `50%` |
| `width` | 数字(px) 或 CSS 尺寸 | `50%` | `align` 非 `full` 时的容器宽度 |

Carousel 专属：

| 选项 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `carouselShowThumbnails` | 布尔 | `false` | `true` 时显示缩略图条，否则显示圆点按钮 |
| `carouselBackground` | CSS 颜色 | 主题次级背景 | 轮播外框背景 |
| `carouselHeight` | 数字(px) 或 CSS 尺寸 | `24rem` | 图片舞台高度 |

Custom 专属：

| 选项 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `grid` | 多行字符串 | 无 | ASCII 网格定义（见 §3.5） |

### 3.4 预置网格定义（CSS Grid，必须逐字匹配）

| 布局 | 所需图片数 | `grid-template-columns` | `grid-template-areas` |
| --- | --- | --- | --- |
| `a` | 2 | `1fr 1fr` | `"image-0 image-1"` |
| `b` | 2 | `2fr 1fr` | `"image-0 image-1"` |
| `c` | 2 | `1fr 2fr` | `"image-1 image-0"` |
| `d` | 3 | `2fr 1fr` | `"image-0 image-1" "image-0 image-2"` |
| `e` | 3 | `1fr 2fr` | `"image-1 image-0" "image-2 image-0"` |
| `f` | 4 | `3fr 1fr` | `"image-0 image-1" "image-0 image-2" "image-0 image-3"` |
| `g` | 4 | `1fr 3fr` | `"image-1 image-0" "image-2 image-0" "image-3 image-0"` |
| `h` | 3 | `1fr 1fr 1fr` | `"image-0 image-1 image-2"` |
| `i` | 4 | `1fr 1fr 1fr 1fr` | `"image-0 image-1 image-2 image-3"` |
| `single` | 1 | `1fr` | `"image-0"` |

> 注意 `c`、`e`、`g` 的 area 中 **`image-1` 在左**（第一张图放大在右），与直觉相反，必须照抄。
> 每张图通过 `grid-area: image-N` 定位（CSS 类 `.image-layouts-image-N { grid-area: image-N; }`）。

### 3.5 Custom Grid（`layout: custom`）

`grid` 为多行 ASCII 图，**空白分隔的 token**；相同 token 合并为一个区域；`.` 表示空单元格；token 按**首次出现顺序**映射到 `image-0`、`image-1`…：

```
grid: |
  A A B
  A A C
```

等价于：

```
grid-template-columns: repeat(3, 1fr);
grid-template-areas:
  "image-0 image-0 image-1"
  "image-0 image-0 image-2";
```

**校验规则（必须实现，非法时报可读错误而非静默渲染）**：

1. 每行 token 数必须相同；
2. 至少 1 个非 `.` token；
3. 至多 20 个不同 token；
4. 每个 token 的单元格必须构成**实心矩形**（CSS grid 强制要求；否则浏览器丢弃整个模板导致图片叠放）。

### 3.6 完整示例

````
```image-layout
---
layout: d
caption: A day at the beach
overlay: always
fit: cover
---
![[beach-1.jpg|Low tide]]
![[beach-2.jpg|Running in the sand]]
![[beach-3.jpg|Sunset]]
```

```image-layout-a
---
descriptions:
  - Under Sail
  - Our spot for the night
---
![Sailing](https://images.unsplash.com/photo-xxx)
![Anchoring](https://images.unsplash.com/photo-yyy)
```

```image-layout-masonry-3
---
fromFolder: Holidays/2024
sortBy: mtime
reverse: true
limit: 12
---

```image-layout
---
layout: custom
grid: |
  A A B
  A A C
---
![[hero.jpg]]
![[detail-1.jpg]]
![[detail-2.jpg]]
```

```image-layout
---
layout: carousel
carouselShowThumbnails: true
carouselHeight: 60vh
---
![[sunset.jpg|Sunset on the sea]]
![[anchorage.jpg|Our spot for the night]]
```
````

---

## 4. 渲染行为规范

以下为**必须匹配的行为契约**，任何实现路线都不得偏离：

### 4.1 图片收集与顺序

1. 解析 front matter → 得到 `data`；剩余正文为 `content`。
2. 从 `content` 逐行收集以 `!` 开头的图片行（顺序 = 出现顺序）。
3. 若 `fromFolder` 存在，追加文件夹图片（直接子文件，扩展名白名单：`avif bmp gif jpeg jpg png svg webp`，按 `name`（localeCompare）或 `mtime` 排序，`reverse` 反转，`limit` 截断）。
4. **块内图片在前，文件夹图片在后**。
5. YAML 解析失败时：`data` 视为空，正文按原始内容处理（图片行仍渲染，不崩溃）。

### 4.2 图片填充 / 截断（padToSlots）

- 图片数 < 布局所需数 → 用**占位图**填充至所需数。
- 图片数 > 所需数 → **截断**到所需数（多余隐藏）。
- 默认占位图（内置，离线可用）：

```
data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" width="640" height="480"><rect width="100%" height="100%" fill="#88888822"/><circle cx="240" cy="170" r="36" fill="#88888855"/><path d="M120 360l110-140 80 95 60-60 130 105z" fill="#88888855"/></svg>
```

（URL 编码后使用；也可配置自定义 URL。）

### 4.3 Overlay（图片描述）

- 描述来源优先级：`descriptions[i]` > 图片行自带描述（wiki 链接 `|` 段 / markdown alt）。
- `overlay: never` → 不渲染 overlay（但 `alt` 属性仍应保留描述）。
- `overlay: always` → 常显。
- `overlay: hover`（默认）→ 默认透明，容器 hover 时显示。
- 样式：图片底部、水平铺满的**半透明白色圆角条**（`rgba(255,255,255,.75)` + `backdrop-filter: blur`），文字居中、`14px`、`font-weight: 500`、深灰字（#111827）。见 §9 CSS。

### 4.4 Fit / Align / Width（网格布局）

| `fit` | 行为 |
| --- | --- |
| `cover`（默认） | `img { width:100%; height:100%; object-fit: cover; object-position: center; }` |
| `contain` | 同上但 `object-fit: contain`（留边） |
| `natural` | `img { height:auto; object-fit: unset; }`，且网格容器 `align-items: start`（每行按图片自身比例） |

`align`（`left`/`center`/`right`/`full`）：非 `full` 时，用 flex 容器包住布局（`justify-content: flex-start/center/flex-end`），布局容器宽度 `50%` 或 `width` 选项（数字 → `Npx`，字符串原样）。

### 4.5 Masonry 分列算法

**不是真正的瀑布流（不等高错位），而是等宽列内顺序堆叠**：

- `index % columns` 决定图片归属列；
- 每列是 `flex-direction: column`、gap `0.5rem`；
- 列之间 gap `0.5rem`；列布局为 `grid-template-columns: repeat(N, 1fr)`。

### 4.6 Carousel

- 默认高度 `24rem`，背景为主题次级背景色（站点中可用 `#f1f5f9` 类浅灰或主题变量）。
- 默认 `carouselShowThumbnails: false` → 底部圆点按钮（pill）；`true` → 缩略图条（可横向滚动）。
- 左右箭头（SVG 图标）循环切换；**键盘 ←/→ 可导航**；容器可 `tabindex` 聚焦。
- 当前图片描述（`descriptions[i]` > alt）显示在舞台下方居中；块级 `caption` 显示在整个轮播下方。
- 循环行为：最后一张 next → 第一张；第一张 prev → 最后一张。

### 4.7 Caption（块级）

- 渲染在**整个布局下方**（网格 / 瀑布流 / 轮播 / 自定义网格都适用）。
- 样式：居中、`12px`、弱化色（Obsidian 用 `--text-muted`，站点可用 `#6b7280`）。
- 空串或不设置 → 不输出。

---

## 5. Obsidian 依赖剥离清单与源码映射

插件代码分三类，迁移时必须分类处理：

### 5.1 可直接复用的纯逻辑（无 Obsidian 依赖）

| 源码文件 | 内容 | 迁移策略 |
| --- | --- | --- |
| `src/utils/images.ts` | 图片行解析（wiki / markdown / 尺寸 / alt） | **直接复制**（纯正则） |
| `src/utils/custom-grid.ts` | ASCII grid → CSS areas + 校验 | **直接复制** |
| `src/utils/front-matter.ts` | YAML front matter 解析/序列化 | 复制（依赖 `front-matter` npm 包，可换成 `gray-matter` / `js-yaml`） |
| `src/utils/options.ts` | 选项归一化 | 直接复制 |
| `src/utils/overlay.ts` | overlay 模式解析 | 直接复制 |
| `src/utils/placeholder.ts` | 占位图 + `padToSlots` | 复制（去掉 Obsidian 设置部分） |
| `src/utils/blocks.ts` | fence 工具 / 布局名解析（`parseMasonryLayoutName` 等） | 按需复制（编辑回写函数不需要） |
| `src/interfaces.ts` | 布局 → 网格模板映射表（`layoutImages`、`layoutTemplates`） | **直接复制**（本指南 §3.4 的数据来源） |

### 5.2 依赖 Obsidian API、必须替换

| 源码文件 | Obsidian 依赖 | 替换方案（静态站点） |
| --- | --- | --- |
| `src/utils/image-resolver.ts` | `metadataCache.getFirstLinkpathDest`、`vault.adapter.getResourcePath`、`Platform.resourcePathPrefix` | wiki 路径直接映射为站点图片路径（见 §7.3 / §8.2 的路径映射规则） |
| `src/utils/folder-images.ts` | `vault.getAbstractFileByPath`、`TFile` | 构建期扫描文件系统 / Hugo `resources` |
| `src/utils/block-images.ts` | `MarkdownPostProcessorContext` | 逻辑保留（块内图片 + fromFolder 合并），实现换为纯函数 |
| `src/main.ts`、`src/processors/*.ts` | 插件注册机制 | 改为构建期调用 / 模板 partial |
| 全部 `src/components/*.svelte` | Svelte + Obsidian 渲染 | 重写为静态 HTML/CSS（见 §7 / §9） |
| `src/views/*`（设置 / 布局选择器） | Obsidian UI | **整块剥离，不需要** |

### 5.3 编辑态交互，整块剥离（静态站点不需要）

| 模块 | 说明 |
| --- | --- |
| `LayoutPicker.svelte`、`LayoutSchematic.svelte` | 空块的交互式布局选择器 |
| `SwitchableLayout.svelte`、`PickerButton` | 已渲染布局上的「切换布局」按钮 |
| `editor-writeback.ts`、`blocks.ts` 中 `updateLayoutInBlockSource` 等 | 把选择写回编辑器 |
| `settings.ts`、`picker-modal.ts` | 设置面板与命令面板 |
| `DropHandler.svelte` | 拖放（Roadmap 未完成项） |

---

## 6. 迁移方案论证与选型

### 6.1 候选方案对比

| 方案 | 语法零改动 | 构建期静态渲染 | SEO | 实现成本 | 适用 |
| --- | --- | --- | --- | --- | --- |
| **A. Hugo Code Block Render Hooks** | ✅ | ✅ | ✅ | 中（Go 模板） | **标准 Hugo ≥ 0.127（推荐）** |
| **B. Node/TS 构建期预处理器** | ✅ | ✅ | ✅ | 中（可直接复用插件 TS 逻辑） | **自研 / 任何 SSG（推荐）** |
| C. Hugo Shortcodes | ❌ 需转换语法 | ✅ | ✅ | 低 | Hugo 老版本 / 接受改语法 |
| D. 客户端 JS 运行时渲染 | ✅ | ❌ 需要 JS | ❌ 首屏无内容 | 低（复用 `test-vault/publish.js`） | 仅作 Carousel 等交互增强 |

### 6.2 决策规则

1. 项目是**标准 Hugo 且版本 ≥ 0.127** → **方案 A**（零额外构建步骤，fence 原样进入模板渲染）。
2. 项目是**自研 / 其它 SSG（Zola、Hexo、Astro 等）** → **方案 B**（逻辑保真度最高，插件的纯逻辑直接复用，产出任意 SSG 都能嵌入的 HTML + CSS）。
3. **Carousel** 需要交互（翻页/缩略图）→ 无论 A 还是 B，都在页面加载后叠加**方案 D 的轻量 JS**（见 §9.2），静态 HTML 兜底显示第一张。
4. 若目标 Hugo 版本 < 0.127 且不愿升级 → 方案 C（Shortcode），并提供「Obsidian 语法 → Shortcode」的一次性转换脚本思路（§8.4）。

> **推荐默认**：方案 A（Hugo）或方案 B（自研），文档已给出完整实现代码，可两者皆备 —— CSS 与 Carousel JS 为共享资产（§9）。

---

## 7. 方案 A：Hugo Code Block Render Hooks

### 7.1 原理与前提

Hugo 从 **v0.127** 起支持 code block render hooks：在 `layouts/_default/_markup/` 下放置 `render-codeblock-<language>.html`，该语言的围栏代码块将不再走默认 `<pre><code>` 渲染，而是交给你的模板。模板内可访问：

- `.Type`：围栏语言（如 `image-layout-a`）
- `.Inner`：代码块内容（字符串，不含 fence）
- `.Attributes`：代码块属性

**前提**：Hugo ≥ 0.127；若主题已自带 `render-codeblock.html`，本方案不覆盖它（用下面的「每语言薄文件」方式，互不冲突）。

### 7.2 文件布局

```
layouts/
  _default/
    _markup/
      render-codeblock-image-layout.html        ← modern 语法（```image-layout）
      render-codeblock-image-layout-a.html      ← legacy（以下为薄文件，见 7.5）
      render-codeblock-image-layout-b.html
      ... （a~i、single、left、center、right、masonry-2~6，共 17 个）
  partials/
    image-layouts/
      render.html                               ← 核心渲染（全部逻辑在此）
  css/ 或 static/css/
      image-layouts.css                         ← §9.1 的样式资产
static/js/
      image-layouts-carousel.js                 ← §9.2（可选，仅 carousel）
```

### 7.3 图片路径映射规则（wiki 链接）

Obsidian 中 `![[beach.jpg]]` 的路径相对仓库根。迁移约定：

- 图片文件放置：博客 `static/images/<vault相对路径>`（或 Hugo `assets/`，用 `resources.GetMatch` 处理）。
- 转换规则：`![[foo/bar.jpg|desc]]` → `<img src="/images/foo/bar.jpg" alt="desc">`。
- 若路径带 Obsidian 百分号编码（`Test%20folder/img.jpg`）→ 解码后映射。
- 远程 `http(s)` 链接：原样输出。
- `fromFolder`：在模板中读取站点根 `static/images/<folder>` 目录（可用 `readDir` 函数扫描，仅直接子文件，扩展名白名单同 §4.1），排序规则同 §4.1。

### 7.4 核心模板（partials/image-layouts/render.html）

> 以下为完整可运行模板。若项目是标准 Hugo 可直接使用；代码中 `$IMG_ROOT` 按你的图片目录调整。

```go-html-template
{{- /*
    image-layouts 核心渲染器。
    接收 .Type（围栏语言）与 .Inner（代码块内容）。
*/ -}}
{{- $lang := .Type -}}
{{- $inner := .Inner -}}
{{- $IMG_ROOT := "/images/" -}}

{{- /* ---------- 1. Legacy 布局名提取（image-layout-<suffix>） ---------- */ -}}
{{- $legacyLayout := "" -}}
{{- $defaultAlign := "full" -}}
{{- if hasPrefix $lang "image-layout-" -}}
  {{- $suffix := strings.TrimPrefix "image-layout-" $lang -}}
  {{- if eq $suffix "left" -}}    {{- $legacyLayout = "single" -}} {{- $defaultAlign = "left" -}}
  {{- else if eq $suffix "center" -}}{{- $legacyLayout = "single" -}} {{- $defaultAlign = "center" -}}
  {{- else if eq $suffix "right" -}}   {{- $legacyLayout = "single" -}} {{- $defaultAlign = "right" -}}
  {{- else -}}                    {{- $legacyLayout = $suffix -}}
  {{- end -}}
{{- end -}}

{{- /* ---------- 2. 解析 front matter 与正文 ---------- */ -}}
{{- $data := dict -}}
{{- $content := $inner -}}
{{- $trimmed := trim $inner "\n" -}}
{{- if hasPrefix $trimmed "---" -}}
  {{- $rest := strings.TrimPrefix "---" $trimmed -}}
  {{- $parts := split $rest "\n---" -}}
  {{- if ge (len $parts) 2 -}}
    {{- $yaml := strings.TrimPrefix "\n" (index $parts 0) -}}
    {{- with transform.Unmarshal (dict "format" "yaml") $yaml -}}
      {{- $data = . -}}
    {{- end -}}
    {{- $content = strings.TrimPrefix "\n" (index $parts 1) -}}
  {{- end -}}
{{- end -}}

{{- /* ---------- 3. 确定 layout（modern 从 data 读取） ---------- */ -}}
{{- $layout := $legacyLayout -}}
{{- if eq $lang "image-layout" -}}
  {{- $layout = index $data "layout" | default "" -}}
{{- end -}}
{{- if hasPrefix $layout "legacy-layout-" -}}{{ $layout = strings.TrimPrefix "legacy-layout-" $layout }}{{- end -}}
{{- if hasPrefix $layout "legacy-masonry-" -}}{{ $layout = printf "masonry-%s" (strings.TrimPrefix "legacy-masonry-" $layout) }}{{- end -}}

{{- /* ---------- 4. 提取图片行 ---------- */ -}}
{{- $images := slice -}}
{{- range (split $content "\n") -}}
  {{- $line := trim . "\r" -}}
  {{- $l := trim $line " \r" -}}
  {{- if hasPrefix $l "!" -}}
    {{- if hasPrefix $l "![[" -}}
      {{- /* wikilink：![[path|desc|300|300x200]] */ -}}
      {{- $emb := strings.TrimSuffix "]]" (strings.TrimPrefix "![[" $l) -}}
      {{- $segs := split $emb "|" -}}
      {{- $path := strings.Trim (index $segs 0) " \t" -}}
      {{- $alt := "" -}}
      {{- range $i, $seg := $segs -}}
        {{- if gt $i 0 -}}
          {{- $s := trim $seg " " -}}
          {{- if not (findRE `^\d+(x\d+)?$` $s) -}}{{- $alt = printf "%s|%s" $alt $s -}}{{- end -}}
        {{- end -}}
      {{- end -}}
      {{- $alt = strings.TrimPrefix "|" $alt -}}
      {{- $images = $images | append (dict "src" (printf "%s%s" $IMG_ROOT $path) "alt" $alt) -}}
    {{- else -}}
      {{- /* markdown：![alt](url)；http(s) 远程原样，本地路径加图片根前缀 */ -}}
      {{- $alt := replaceRE `^!\s*\[([^\]]*)\]\(.*\)$` "$1" $l -}}
      {{- $url := replaceRE `^!\s*\[[^\]]*\]\(([^()]*)\)$` "$1" $l -}}
      {{- if hasPrefix (lower $url) "http" -}}
        {{- $images = $images | append (dict "src" $url "alt" $alt) -}}
      {{- else -}}
        {{- $images = $images | append (dict "src" (printf "%s%s" $IMG_ROOT $url) "alt" $alt) -}}
      {{- end -}}
    {{- end -}}
  {{- end -}}
{{- end -}}

{{- /* ---------- 4.5 fromFolder：扫描 static/images/<folder> 直接子图片文件，排在块内图片之后 ---------- */ -}}
{{- /* 注：按文件名排序 + reverse + limit 在 v0.127 即可实现；mtime 排序需要 os.Stat（Hugo ≥ 0.128），本模板回退为按名排序，如必须 mtime 请用方案 B */ -}}
{{- $folder := index $data "fromFolder" -}}
{{- if $folder -}}
  {{- with readDir (printf "static/images/%s" $folder) -}}
    {{- $names := slice -}}
    {{- range . -}}
      {{- if and (not .IsDir) (findRE `(?i)\.(avif|bmp|gif|jpeg|jpg|png|svg|webp)$` .Name) }}{{ $names = $names | append .Name }}{{ end -}}
    {{- end -}}
    {{- $names = sort $names -}}
    {{- if eq (index $data "reverse") true }}{{ $names = collections.Reverse $names }}{{ end -}}
    {{- $limit := index $data "limit" -}}
    {{- if and (ne $limit nil) (gt (len $names) $limit) }}{{ $names = first $limit $names }}{{ end -}}
    {{- range $names -}}{{- $images = $images | append (dict "src" (printf "/images/%s/%s" $folder .) "alt" "") -}}{{- end -}}
  {{- end -}}
{{- end -}}

{{- /* ---------- 5. 选项归一化 ---------- */ -}}
{{- $caption := index $data "caption" | default "" -}}
{{- $descriptions := index $data "descriptions" | default (slice) -}}
{{- $overlay := "hover" -}}
{{- if index $data "overlay" -}}
  {{- $overlay = index $data "overlay" -}}
{{- else if eq (index $data "permanentOverlay") true -}}
  {{- $overlay = "always" -}}
{{- end -}}
{{- $fit := index $data "fit" | default "cover" -}}
{{- $align := index $data "align" | default $defaultAlign -}}
{{- $width := index $data "width" | default "50%" -}}
{{- if findRE `^\d+$` $width }}{{ $width = printf "%spx" $width }}{{ end -}}

{{- /* ---------- 6. 占位图（与插件内置一致；用 urlquery 编码，避免手写转义出错） ---------- */ -}}
{{- $ph := printf "data:image/svg+xml,%s" (urlquery "<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"640\" height=\"480\"><rect width=\"100%\" height=\"100%\" fill=\"#88888822\"/><circle cx=\"240\" cy=\"170\" r=\"36\" fill=\"#88888855\"/><path d=\"M120 360l110-140 80 95 60-60 130 105z\" fill=\"#88888855\"/></svg>") -}}

{{- /* ---------- 7. 各布局渲染 ---------- */ -}}

{{- /* 7.1 carousel：输出静态首图 + 挂载容器（JS 增强见 §9.2） ---------- */ -}}
{{- if eq $layout "carousel" -}}
  {{- $h := index $data "carouselHeight" | default "24rem" -}}
  {{- if findRE `^\d+$` $h }}{{ $h = printf "%spx" $h }}{{ end -}}
  {{- $bg := index $data "carouselBackground" | default "#f1f5f9" -}}
  <div class="image-layout-carousel" data-thumbnails="{{ cond (eq (index $data "carouselShowThumbnails") true) "true" "false" }}" style="{{ printf "background:%s" $bg | safeCSS }}">
    <div class="slides-container" style="{{ printf "height:%s" $h | safeCSS }}">
      {{- range $i, $img := $images -}}
        <div class="slide{{ if eq $i 0 }} active{{ end }}">
          <img src="{{ index $img "src" }}" alt="{{ index $img "alt" | default (printf "Image %d" (add $i 1)) }}">
          {{- with index $descriptions $i }}<div class="slide-caption">{{ . }}</div>{{ else }}{{ with index $img "alt" }}<div class="slide-caption">{{ . }}</div>{{ end }}{{ end }}
        </div>
      {{- end -}}
    </div>
    <button class="nav-button prev" type="button" aria-label="上一张">‹</button>
    <button class="nav-button next" type="button" aria-label="下一张">›</button>
  </div>
  {{- with $caption }}<div class="image-layouts-caption">{{ . }}</div>{{ end -}}
  {{- return -}}
{{- end -}}

{{- /* 7.2 custom：ASCII grid → CSS areas（校验规则见 §3.5） ---------- */ -}}
{{- if eq $layout "custom" -}}
  {{- $gridSpec := index $data "grid" | default "" -}}
  {{- /* 行以字符串存储（Hugo 的 append 会展开 slice 参数，故不直接存 [][]string） */ -}}
  {{- $rows := slice -}}
  {{- range (split $gridSpec "\n") -}}
    {{- $t := trim . " \t" -}}
    {{- if ne $t "" }}{{ $rows = $rows | append $t }}{{ end -}}
  {{- end -}}
  {{- if eq (len $rows) 0 -}}<p class="image-layouts-error">Image Layouts: A custom layout needs a `grid` option with rows of letters.</p>{{- return -}}{{- end -}}
  {{- $cols := len (split (index $rows 0) " ") -}}
  {{- /* 每行 cell 数一致性检查（§3.5 规则 1） */ -}}
  {{- range $rows -}}{{- if ne (len (split . " ")) $cols -}}<p class="image-layouts-error">Image Layouts: Every row in `grid` must have the same number of cells.</p>{{- return -}}{{- end -}}{{- end -}}
  {{- $order := slice -}}
  {{- range $rows -}}
    {{- range (split . " ") -}}
      {{- if and (ne . ".") (not (in $order .)) }}{{ $order = $order | append . }}{{ end -}}
    {{- end -}}
  {{- end -}}
  {{- /* grid-template-areas 直接使用 token 名（效果等价插件的 image-N 编号：token 按首次出现顺序对应第 N 张图，且 cell 用同名 grid-area，互不冲突） */ -}}
  {{- $areasLines := slice -}}
  {{- range $rows -}}
    {{- $cells := slice -}}
    {{- range (split . " ") -}}{{ $cells = $cells | append . }}{{ end -}}
    {{- $areasLines = $areasLines | append (printf "\"%s\"" (delimit $cells " ")) -}}
  {{- end -}}
  {{- $areas := delimit $areasLines " " -}}
  <div class="image-layouts image-layouts-grid image-layouts-custom" style="{{ printf "grid-template-columns: repeat(%d, 1fr); grid-template-areas: %s;" $cols $areas | safeCSS }}">
    {{- range $i, $token := $order -}}
      {{- $img := index $images $i | default (dict "src" $ph "alt" "") -}}
      {{- $desc := index $descriptions $i | default (index $img "alt") -}}
      <div class="image-layouts-image-cell" style="{{ printf "grid-area: %s" $token | safeCSS }}">
        <img src="{{ index $img "src" }}" alt="{{ $desc | default (printf "Image %d" (add $i 1)) }}" loading="lazy">
        {{- if and $desc (ne $overlay "never") -}}
        <div class="image-layouts-overlay{{ if ne $overlay "always" }} image-layouts-overlay-hidden{{ end }}"><div class="image-layouts-overlay-text">{{ $desc }}</div></div>
        {{- end -}}
      </div>
    {{- end -}}
  </div>
  {{- with $caption }}<div class="image-layouts-caption">{{ . }}</div>{{ end -}}
  {{- return -}}
{{- end -}}

{{- /* 7.3 masonry：N 列顺序堆叠 ---------- */ -}}
{{- if findRE `^masonry-[2-6]$` $layout -}}
  {{- $cols := int (replaceRE `^masonry-([2-6])$` "$1" $layout) -}}
  {{- /* 占位填充至图片数？masonry 无固定 slots —— 插件行为：所有图片展示，不填充 */ -}}
  <div class="image-layouts-masonry-grid-{{ $cols }}">
    {{- range seq 0 (sub $cols 1) -}}
      {{- $col := . -}}
      <div class="image-layouts-masonry-column">
        {{- range $i, $img := $images -}}
          {{- if eq (mod $i $cols) $col -}}
            {{- $desc := index $descriptions $i | default (index $img "alt") -}}
            <div class="image-layouts-image-cell">
              <img src="{{ index $img "src" }}" alt="{{ $desc | default (printf "Image %d" (add $i 1)) }}" loading="lazy">
              {{- if and $desc (ne $overlay "never") -}}
              <div class="image-layouts-overlay{{ if ne $overlay "always" }} image-layouts-overlay-hidden{{ end }}"><div class="image-layouts-overlay-text">{{ $desc }}</div></div>
              {{- end -}}
            </div>
          {{- end -}}
        {{- end -}}
      </div>
    {{- end -}}
  </div>
  {{- with $caption }}<div class="image-layouts-caption">{{ . }}</div>{{ end -}}
  {{- return -}}
{{- end -}}

{{- /* 7.4 预置网格（a~i / single） ---------- */ -}}
{{- $grid := dict "a" (dict "cols" "1fr 1fr" "areas" "\"image-0 image-1\"" "slots" 2) "b" (dict "cols" "2fr 1fr" "areas" "\"image-0 image-1\"" "slots" 2) "c" (dict "cols" "1fr 2fr" "areas" "\"image-1 image-0\"" "slots" 2) "d" (dict "cols" "2fr 1fr" "areas" "\"image-0 image-1\" \"image-0 image-2\"" "slots" 3) "e" (dict "cols" "1fr 2fr" "areas" "\"image-1 image-0\" \"image-2 image-0\"" "slots" 3) "f" (dict "cols" "3fr 1fr" "areas" "\"image-0 image-1\" \"image-0 image-2\" \"image-0 image-3\"" "slots" 4) "g" (dict "cols" "1fr 3fr" "areas" "\"image-1 image-0\" \"image-2 image-0\" \"image-3 image-0\"" "slots" 4) "h" (dict "cols" "1fr 1fr 1fr" "areas" "\"image-0 image-1 image-2\"" "slots" 3) "i" (dict "cols" "1fr 1fr 1fr 1fr" "areas" "\"image-0 image-1 image-2 image-3\"" "slots" 4) "single" (dict "cols" "1fr" "areas" "\"image-0\"" "slots" 1) -}}
{{- $g := index $grid $layout -}}
{{- if not $g -}}
  <p class="image-layouts-error">Image Layouts: unknown layout "{{ $layout }}"</p>
{{- else -}}
  {{- $slots := index $g "slots" -}}
  {{- /* 填充占位 */ -}}
  {{- range seq $slots -}}
    {{- if lt (len $images) $slots }}{{ $images = $images | append (dict "src" $ph "alt" "") }}{{ end -}}
  {{- end -}}
  {{- $images = first $slots $images -}}
  <div class="image-layouts-align image-layouts-align-{{ $align }}">
    <div class="image-layouts image-layouts-grid image-layouts-layout-{{ $layout }} image-layouts-fit-{{ $fit }}" style="{{ printf "grid-template-columns: %s; grid-template-areas: %s;%s" (index $g "cols") (index $g "areas") (cond (ne $align "full") (printf " width:%s; max-width:%s" $width $width) "") | safeCSS }}">
      {{- range $i, $img := $images -}}
        {{- $desc := index $descriptions $i | default (index $img "alt") -}}
        <div class="image-layouts-image-{{ $i }} image-layouts-image-cell">
          <img src="{{ index $img "src" }}" alt="{{ $desc | default (printf "Image %d" (add $i 1)) }}" loading="lazy">
          {{- if and $desc (ne $overlay "never") -}}
          <div class="image-layouts-overlay{{ if ne $overlay "always" }} image-layouts-overlay-hidden{{ end }}"><div class="image-layouts-overlay-text">{{ $desc }}</div></div>
          {{- end -}}
        </div>
      {{- end -}}
    </div>
  </div>
  {{- with $caption }}<div class="image-layouts-caption">{{ . }}</div>{{ end -}}
{{- end -}}
```

> **说明**：模板中 custom 分支为简化实现（area 名直接用 token，未做「实心矩形校验」）。若需完整校验，请以 `src/utils/custom-grid.ts`（或 `test-vault/publish.ts` 的 `createCustomGridLayout`）为准，在模板前置阶段用 `transform.Unmarshal` 解析后校验；或 custom 布局交给方案 B 的 Node 脚本处理（§8.2 已含完整校验）。

### 7.5 薄文件与生成命令

`render-codeblock-image-layout.html`（modern）：

```go-html-template
{{ partial "image-layouts/render.html" . }}
```

17 个 legacy 薄文件内容完全相同（仅文件名不同），用命令生成：

```bash
cd <hugo-project>
mkdir -p layouts/_default/_markup layouts/partials/image-layouts
for L in a b c d e f g h i single left center right \
         masonry-2 masonry-3 masonry-4 masonry-5 masonry-6; do
  printf '{{ partial "image-layouts/render.html" . }}\n' \
    > "layouts/_default/_markup/render-codeblock-image-layout-${L}.html"
done
printf '{{ partial "image-layouts/render.html" . }}\n' \
  > "layouts/_default/_markup/render-codeblock-image-layout.html"
```

> 这样**不会**覆盖主题的 `render-codeblock.html`，普通代码块（```go、```python 等）完全不受影响。

### 7.6 配置（hugo.toml）

```toml
[markup.goldmark.renderer]
  unsafe = true   # 可选：若内容含原始 HTML（方案 B 输出 HTML 时必需）
```

---

## 8. 方案 B：Node/TS 预处理器

### 8.1 原理

构建前运行一个 Node 脚本，扫描 Markdown 文件，把 `` ```image-layout* `` 代码块**原地替换**为 HTML 片段（嵌入 markdown 或直接输出 .html），并复制一份 CSS。任何 SSG（Hugo / Zola / Hexo / Astro / 自研）都能嵌入，因为渲染结果就是纯 HTML。

- 逻辑保真度最高：插件的 `images.ts`、`custom-grid.ts`、`front-matter.ts` 等**纯 TS 逻辑直接复制**（见 §5.1），无需用模板语言重写。
- 图片路径映射：与 §7.3 相同；`fromFolder` 用 Node `fs.readdir` 实现（直接子文件 + 扩展名白名单 + 排序）。

### 8.2 脚本骨架（src/image-layouts/transform.mjs）

> 以下为可运行的完整实现（基于插件源码纯逻辑移植，依赖仅 `gray-matter` 或手写解析）。放置于目标项目 `scripts/` 下。

```js
// scripts/image-layouts.mjs
// 用法：node scripts/image-layouts.mjs <markdown-file> [--out <html-dir>]
// 或作为库：transformMarkdown(markdown, { imageRoot: "/images/" })

import fs from "node:fs";
import path from "node:path";

// ---------- 1. 图片行解析（移植自 src/utils/images.ts，纯正则） ----------
const regexWiki = /\[\[([^\]]+)\]\]/;
const regexMd = /\[([^\]]*)\]\(([^()]*(?:\([^()]*\)[^()]*)*)\)/;
const regexWikiGlobal = /\[\[([^\]]*)\]\]/g;

function safeDecode(link) {
  try { return decodeURIComponent(link); } catch { return link; }
}

function parsePipeSegments(segments) {
  const attrs = { alt: undefined, width: undefined, height: undefined };
  const altParts = [];
  for (const segment of segments) {
    const trimmed = segment.trim();
    const size = trimmed.match(/^(\d+)(?:x(\d+))?$/);
    if (size) {
      attrs.width = Number(size[1]);
      if (size[2]) attrs.height = Number(size[2]);
    } else if (trimmed) {
      altParts.push(trimmed);
    }
  }
  if (altParts.length > 0) attrs.alt = altParts.join("|");
  return attrs;
}

export function getImageFromLine(line) {
  const mdMatch = line.match(regexMd);
  if (mdMatch) {
    const altRaw = mdMatch[1] ?? "";
    const link = mdMatch[2];
    if (link) {
      const attrs = parsePipeSegments(altRaw.split("|"));
      if (link.toLowerCase().startsWith("http")) return { type: "external", link, ...attrs };
      return { type: "local", link, ...attrs };
    }
  } else if (line.match(regexWikiGlobal)) {
    const embed = line.match(regexWiki)?.[1];
    if (embed) {
      const [link, ...pipeSegments] = embed.split("|");
      const attrs = parsePipeSegments(pipeSegments);
      if (link.trim()) return { type: "local", link: safeDecode(link.trim()), ...attrs };
    }
  }
  return null;
}

export function getImages(source) {
  return source
    .split("\n")
    .filter((row) => row.startsWith("!"))
    .map((line) => getImageFromLine(line))
    .filter(Boolean);
}

// ---------- 2. 布局定义（移植自 src/interfaces.ts） ----------
const layoutImages = { a: 2, b: 2, c: 2, d: 3, e: 3, f: 4, g: 4, h: 3, i: 4, single: 1 };
const layoutTemplates = {
  a: { columns: "1fr 1fr", areas: '"image-0 image-1"' },
  b: { columns: "2fr 1fr", areas: '"image-0 image-1"' },
  c: { columns: "1fr 2fr", areas: '"image-1 image-0"' },
  d: { columns: "2fr 1fr", areas: '"image-0 image-1" "image-0 image-2"' },
  e: { columns: "1fr 2fr", areas: '"image-1 image-0" "image-2 image-0"' },
  f: { columns: "3fr 1fr", areas: '"image-0 image-1" "image-0 image-2" "image-0 image-3"' },
  g: { columns: "1fr 3fr", areas: '"image-1 image-0" "image-2 image-0" "image-3 image-0"' },
  h: { columns: "1fr 1fr 1fr", areas: '"image-0 image-1 image-2"' },
  i: { columns: "1fr 1fr 1fr 1fr", areas: '"image-0 image-1 image-2 image-3"' },
  single: { columns: "1fr", areas: '"image-0"' },
};

// ---------- 3. custom grid 解析（移植自 src/utils/custom-grid.ts，含校验） ----------
const MAX_SLOTS = 20;
export function parseCustomGrid(spec) {
  if (typeof spec !== "string" || spec.trim() === "") {
    return { error: "A custom layout needs a `grid` option with rows of letters, e.g.\ngrid: |\n  A A B\n  A A C" };
  }
  const rows = spec.split("\n").map((l) => l.trim()).filter((l) => l !== "").map((l) => l.split(/\s+/));
  const columns = rows[0].length;
  if (rows.some((row) => row.length !== columns)) return { error: "Every row in `grid` must have the same number of cells." };
  const order = [];
  for (const row of rows) for (const cell of row) if (cell !== "." && !order.includes(cell)) order.push(cell);
  if (order.length === 0) return { error: "The `grid` needs at least one image cell." };
  if (order.length > MAX_SLOTS) return { error: `\`grid\` supports up to ${MAX_SLOTS} images.` };
  for (const token of order) {
    let minRow = Infinity, maxRow = -1, minCol = Infinity, maxCol = -1, count = 0;
    rows.forEach((row, r) => row.forEach((cell, c) => {
      if (cell === token) { minRow = Math.min(minRow, r); maxRow = Math.max(maxRow, r); minCol = Math.min(minCol, c); maxCol = Math.max(maxCol, c); count++; }
    }));
    if (count !== (maxRow - minRow + 1) * (maxCol - minCol + 1)) return { error: `The cells for "${token}" must form a solid rectangle.` };
  }
  const templateAreas = rows
    .map((row) => `"${row.map((cell) => (cell === "." ? "." : `image-${order.indexOf(cell)}`)).join(" ")}"`)
    .join(" ");
  return { grid: { columns, rows: rows.length, slots: order.length, templateAreas } };
}

// ---------- 4. 占位图（移植自 src/utils/placeholder.ts） ----------
const PLACEHOLDER_DATA_URI = "data:image/svg+xml," + encodeURIComponent(
  '<svg xmlns="http://www.w3.org/2000/svg" width="640" height="480"><rect width="100%" height="100%" fill="#88888822"/><circle cx="240" cy="170" r="36" fill="#88888855"/><path d="M120 360l110-140 80 95 60-60 130 105z" fill="#88888855"/></svg>'
);

function padToSlots(images, slots, placeholderUrl = PLACEHOLDER_DATA_URI) {
  if (images.length >= slots) return images.slice(0, slots);
  return [...images, ...Array(slots - images.length).fill({ type: "external", link: placeholderUrl })];
}

// ---------- 5. 选项归一化（移植自 src/utils/options.ts / overlay.ts） ----------
const normalizeDescriptions = (v) => (Array.isArray(v) ? v.map((i) => (i == null ? undefined : String(i))) : []);
const normalizeAlign = (v, fallback = "full") => ["left", "center", "right", "full"].includes(v) ? v : fallback;
const resolveOverlayMode = (data) => {
  if (data?.overlay && ["never", "hover", "always"].includes(data.overlay)) return data.overlay;
  if (typeof data?.permanentOverlay === "boolean") return data.permanentOverlay ? "always" : "hover";
  return "hover";
};

// ---------- 6. YAML front matter 解析（可用 gray-matter 替代） ----------
function parseFrontMatterBlock(content) {
  const lines = content.split("\n");
  if (lines[0]?.trim() !== "---") return { data: null, body: content };
  const closeIdx = lines.slice(1).findIndex((l) => l.trim() === "---");
  if (closeIdx < 0) return { data: null, body: content };
  const yaml = lines.slice(1, closeIdx + 1).join("\n");
  let data = null;
  try {
    // 目标项目若用 gray-matter：import fm from "gray-matter"; 直接返回
    // 这里给出一个最小 YAML 子集解析（键: 值 + 列表 + 块标量）
    data = parseSimpleYaml(yaml);
  } catch { data = null; }
  return { data, body: lines.slice(closeIdx + 2).join("\n") };
}

// 最小 YAML 解析器（覆盖本插件用到的语法；生产建议换 gray-matter）
function parseSimpleYaml(yaml) {
  const out = {};
  const lines = yaml.split("\n");
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    if (!line.trim() || line.trim().startsWith("#")) continue;
    const m = line.match(/^(\S+?):\s*(.*)$/);
    if (!m) continue;
    const key = m[1];
    let value = m[2].trim().replace(/^["'](.*)["']$/, "$1");
    if (value === "" || value === "|" || /^[|>][+-]?\d*$/.test(value)) {
      // 块标量（grid: |）或列表（descriptions:）或空值：
      // 收集后续缩进行；列表项（- 开头）→ 数组，块标量文本 → 按行拼接
      const raw = [];
      while (i + 1 < lines.length && (/^\s/.test(lines[i + 1]) || lines[i + 1] === "")) {
        i++;
        raw.push(lines[i]);
      }
      const firstTrimmed = raw.find((l) => l.trim() !== "")?.trim() ?? "";
      if (firstTrimmed.startsWith("- ")) {
        out[key] = raw.map((l) => l.trim().replace(/^- /, "").trim().replace(/^["'](.*)["']$/, "$1"));
      } else {
        out[key] = raw.map((l) => l.trim()).filter((l) => l !== "").join("\n");
      }
      continue;
    }
    if (value.startsWith("- ")) {
      const items = [];
      while (i < lines.length) {
        const l = lines[i].trim();
        if (l.startsWith("- ")) items.push(l.slice(2).trim());
        else break;
        i++;
      }
      out[key] = items;
      continue;
    }
    if (value === "true") out[key] = true;
    else if (value === "false") out[key] = false;
    else if (/^-?\d+(\.\d+)?$/.test(value)) out[key] = Number(value);
    else out[key] = value;
  }
  return out;
}

// ---------- 7. fromFolder（Node fs 实现，替代 folder-images.ts） ----------
import { readdirSync } from "node:fs";
const IMAGE_EXTENSIONS = new Set(["avif", "bmp", "gif", "jpeg", "jpg", "png", "svg", "webp"]);
// fsRoot：真实文件系统根（默认 static/images）；urlRoot：站点 URL 前缀（默认 /images/）
function getFolderImages(folderPath, options = {}, fsRoot = "static/images", urlRoot = "/images/") {
  const full = path.join(fsRoot, folderPath);
  let entries = [];
  try { entries = readdirSync(full, { withFileTypes: true }); } catch { return []; }
  const files = entries
    .filter((e) => e.isFile() && IMAGE_EXTENSIONS.has(path.extname(e.name).slice(1).toLowerCase()))
    .map((e) => ({ name: e.name, mtime: fs.statSync(path.join(full, e.name)).mtimeMs }));
  files.sort(options.sortBy === "mtime" ? (a, b) => a.mtime - b.mtime : (a, b) => a.name.localeCompare(b.name));
  if (options.reverse) files.reverse();
  const limited = typeof options.limit === "number" && options.limit > 0 ? files.slice(0, options.limit) : files;
  return limited.map((f) => ({ type: "external", link: `${urlRoot}${folderPath}/${f.name}`.replace(/\/{2,}/g, "/") }));
}

// ---------- 8. HTML 生成 ----------
function esc(s) { return String(s ?? "").replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;"); }

function imageCell(img, index, desc, overlay, fit) {
  const o = overlay !== "never" && desc
    ? `<div class="image-layouts-overlay${overlay !== "always" ? " image-layouts-overlay-hidden" : ""}"><div class="image-layouts-overlay-text">${esc(desc)}</div></div>`
    : "";
  return `<div class="image-layouts-image-cell${fit === "contain" ? " image-layouts-fit-contain" : ""}${fit === "natural" ? " image-layouts-fit-natural" : ""}">
    <img src="${esc(img.link)}" alt="${esc(desc ?? `Image ${index + 1}`)}" loading="lazy">${o}</div>`;
}

function withCaption(html, caption) {
  return caption ? `${html}<div class="image-layouts-caption">${esc(caption)}</div>` : html;
}

function renderBlock(source, options = {}) {
  const { imageRoot = "/images/", layoutHint } = options;
  const parsed = parseFrontMatterBlock(source);
  // data 恒为对象（无 front matter / 空块时为 {}），保证后续 data.xxx 访问安全
  const data = parsed.data ?? {};
  const body = parsed.body;
  const images = getImages(body).map((img) => ({
    ...img,
    link: img.type === "external" ? img.link : imageRoot + img.link,
  }));
  const folderImages = data?.fromFolder ? getFolderImages(data.fromFolder, data, options.fsRoot ?? "static/images", imageRoot) : [];
  const allImages = [...images, ...folderImages];
  const descriptions = normalizeDescriptions(data?.descriptions);

  // legacy fence 的布局由 fence 名决定（layoutHint），优先于块内 layout 字段；
  // modern 语法（无 hint）使用块内 layout 字段。
  const layout = layoutHint ?? (typeof data?.layout === "string" ? data.layout : undefined);
  // 空块（无 layout）：静态站点中渲染为注释（插件中此处是交互式布局选择器）
  if (!layout) return "<!-- image-layout: empty block (no layout specified) -->";
  const normalized = layout?.startsWith("legacy-layout-") ? layout.slice(14)
    : layout?.startsWith("legacy-masonry-") ? `masonry-${layout.slice(15)}` : layout;

  // legacy 对齐简写：image-layout-left/center/right → single + align
  const ALIGN_SHORTHANDS = { left: "left", center: "center", right: "right" };
  const effectiveLayout = ALIGN_SHORTHANDS[normalized] ? "single" : normalized;
  const align = normalizeAlign(data?.align, ALIGN_SHORTHANDS[normalized] ?? "full");
  const fit = data?.fit ?? "cover";
  const width = typeof data?.width === "number" ? `${data.width}px` : data?.width ?? "50%";
  const overlay = resolveOverlayMode(data);

  // carousel
  if (effectiveLayout === "carousel") {
    const h = typeof data.carouselHeight === "number" ? `${data.carouselHeight}px` : (data.carouselHeight ?? "24rem");
    const bg = data.carouselBackground ?? "#f1f5f9";
    const thumbs = data.carouselShowThumbnails ? "true" : "false";
    const slides = allImages.map((img, i) => {
      const d = descriptions[i] ?? img.alt;
      return `<div class="slide${i === 0 ? " active" : ""}"><img src="${esc(img.link)}" alt="${esc(d ?? `Image ${i + 1}`)}">${d ? `<div class="slide-caption">${esc(d)}</div>` : ""}</div>`;
    }).join("");
    return withCaption(
      `<div class="image-layout-carousel" data-thumbnails="${thumbs}" style="background:${esc(bg)}">` +
        `<div class="slides-container" style="height:${esc(h)}">${slides}</div>` +
        `<button class="nav-button prev" type="button" aria-label="上一张">‹</button>` +
        `<button class="nav-button next" type="button" aria-label="下一张">›</button></div>`,
      data.caption
    );
  }

  // custom
  if (effectiveLayout === "custom") {
    const parsed = parseCustomGrid(data?.grid);
    if (parsed.error) return `<p class="image-layouts-error">Image Layouts: ${esc(parsed.error)}</p>`;
    const cells = padToSlots(allImages, parsed.grid.slots).map((img, i) =>
      `<div class="image-layouts-image-cell${fit === "contain" ? " image-layouts-fit-contain" : ""}${fit === "natural" ? " image-layouts-fit-natural" : ""}" style="grid-area: image-${i}">` +
        `<img src="${esc(img.link)}" alt="${esc(descriptions[i] ?? img.alt ?? `Image ${i + 1}`)}" loading="lazy">` +
        (overlay !== "never" && (descriptions[i] ?? img.alt) ? `<div class="image-layouts-overlay${overlay !== "always" ? " image-layouts-overlay-hidden" : ""}"><div class="image-layouts-overlay-text">${esc(descriptions[i] ?? img.alt)}</div></div>` : "") +
      `</div>`).join("");
    return withCaption(
      `<div class="image-layouts image-layouts-grid image-layouts-custom" style="grid-template-columns: repeat(${parsed.grid.columns}, 1fr); grid-template-areas: ${parsed.grid.templateAreas};">${cells}</div>`,
      data.caption
    );
  }

  // masonry
  const masonry = effectiveLayout?.match(/^masonry-([2-6])$/);
  if (masonry) {
    const cols = Number(masonry[1]);
    const columns = Array.from({ length: cols }, () => []);
    allImages.forEach((img, i) => columns[i % cols].push({ img, idx: i })); // 保留全局索引
    const html = `<div class="image-layouts-masonry-grid-${cols}">` + columns.map((col) =>
      `<div class="image-layouts-masonry-column">` + col.map(({ img, idx }) => {
        const d = descriptions[idx] ?? img.alt;
        return `<div class="image-layouts-image-cell"><img src="${esc(img.link)}" alt="${esc(d ?? `Image ${idx + 1}`)}" loading="lazy">` +
          (overlay !== "never" && d ? `<div class="image-layouts-overlay${overlay !== "always" ? " image-layouts-overlay-hidden" : ""}"><div class="image-layouts-overlay-text">${esc(d)}</div></div>` : "") + `</div>`;
      }).join("") + `</div>`).join("") + `</div>`;
    return withCaption(html, data.caption);
  }

  // 预置网格（含 legacy fence 已由调用方归一化）
  const tpl = layoutTemplates[effectiveLayout];
  if (!tpl) return `<p class="image-layouts-error">Image Layouts: unknown layout "${esc(layout)}"</p>`;
  const slots = layoutImages[effectiveLayout];
  const cells = padToSlots(allImages, slots).map((img, i) => {
    const d = descriptions[i] ?? img.alt;
    return `<div class="image-layouts-image-${i} image-layouts-image-cell"><img src="${esc(img.link)}" alt="${esc(d ?? `Image ${i + 1}`)}" loading="lazy">` +
      (overlay !== "never" && d ? `<div class="image-layouts-overlay${overlay !== "always" ? " image-layouts-overlay-hidden" : ""}"><div class="image-layouts-overlay-text">${esc(d)}</div></div>` : "") + `</div>`;
  }).join("");
  const widthStyle = align !== "full" ? ` style="width:${esc(width)};max-width:${esc(width)}"` : "";
  const html = `<div class="image-layouts-align image-layouts-align-${align}">` +
    `<div class="image-layouts image-layouts-grid image-layouts-layout-${effectiveLayout} image-layouts-fit-${fit}" style="grid-template-columns: ${tpl.columns}; grid-template-areas: ${tpl.areas};"${widthStyle}>${cells}</div></div>`;
  return withCaption(html, data.caption);
}

// ---------- 9. Markdown 扫描与替换 ----------
const FENCE_RE = /(`{3,}|~{3,})(image-layout[\w-]*)\s*\n([\s\S]*?)\1/g;

export function transformMarkdown(markdown, options) {
  return markdown.replace(FENCE_RE, (_, fence, lang, source) => {
    // legacy fence（image-layout-a 等）：布局来自 fence 名，作为 layoutHint 传入，
    // 不修改块内容（块内 front matter 保持原样，避免双重 ---）
    const layoutHint = lang === "image-layout" ? undefined : lang.replace(/^image-layout-?/, "");
    return renderBlock(source, { ...options, layoutHint });
  });
}

// ---------- CLI ----------
if (process.argv[1] && import.meta.url.endsWith(process.argv[1].split(/[\\/]/).pop())) {
  const file = process.argv[2];
  const markdown = fs.readFileSync(file, "utf8");
  process.stdout.write(transformMarkdown(markdown, { imageRoot: "/images/" }));
}
```

### 8.3 集成到构建流程

```bash
# 示例：构建前处理 content/ 下所有 md
node scripts/image-layouts.mjs content/posts/foo.md > tmp/foo.md   # 单文件
# 或批量（Windows / bash 二选一）：
find content -name "*.md" | while read f; do node scripts/image-layouts.mjs "$f" > "$f.tmp" && mv "$f.tmp" "$f"; done
```

- 产出 HTML 嵌入 markdown 时，目标 SSG 需允许 raw HTML（Hugo：`markup.goldmark.renderer.unsafe = true`）。
- 或者更稳妥：把脚本接到你的渲染管线中，对**渲染后的 HTML** 做替换（不污染 markdown 源）。
- 同时把 §9.1 的 `image-layouts.css` 复制到站点 `static/css/`，在模板 `<head>` 引入。

### 8.4 Legacy 语法 → Shortcode 的一次性转换（仅方案 C 需要）

若最终选择 Shortcode 路线，转换脚本思路：用 `transformMarkdown` 相同的 fence 正则，把

```
```image-layout-a
...内容...
```
```

替换为

```
{{< image-layout-a >}}
...内容（图片行原样）...
{{< /image-layout-a >}}
```

Shortcode 模板内复用 §7.4 partial 的逻辑即可（shortcode 中 `.Inner` 同样可用）。本指南不展开方案 C，因为 A/B 已覆盖语法零改动的需求。

---

## 9. 共享资产：CSS 与 Carousel JS

### 9.1 image-layouts.css（完整，直接使用）

> 从插件 Svelte 组件 + UnoCSS 提取，类名与插件一致。保存为 `static/css/image-layouts.css`。

```css
/* ===== 容器 ===== */
.image-layouts { display: block; }

/* 网格基础（gap 0.5rem 与原插件一致） */
.image-layouts-grid { display: grid; grid-gap: 0.5rem; }
.image-layouts-grid.image-layouts-fit-natural { align-items: start; }
.image-layouts-custom { grid-gap: 0.5rem; }

/* ===== 预置网格 ===== */
.image-layouts-layout-a { grid-template-columns: 1fr 1fr; grid-template-areas: "image-0 image-1"; }
.image-layouts-layout-b { grid-template-columns: 2fr 1fr; grid-template-areas: "image-0 image-1"; }
.image-layouts-layout-c { grid-template-columns: 1fr 2fr; grid-template-areas: "image-1 image-0"; }
.image-layouts-layout-d { grid-template-columns: 2fr 1fr; grid-template-areas: "image-0 image-1" "image-0 image-2"; }
.image-layouts-layout-e { grid-template-columns: 1fr 2fr; grid-template-areas: "image-1 image-0" "image-2 image-0"; }
.image-layouts-layout-f { grid-template-columns: 3fr 1fr; grid-template-areas: "image-0 image-1" "image-0 image-2" "image-0 image-3"; }
.image-layouts-layout-g { grid-template-columns: 1fr 3fr; grid-template-areas: "image-1 image-0" "image-2 image-0" "image-3 image-0"; }
.image-layouts-layout-h { grid-template-columns: 1fr 1fr 1fr; grid-template-areas: "image-0 image-1 image-2"; }
.image-layouts-layout-i { grid-template-columns: 1fr 1fr 1fr 1fr; grid-template-areas: "image-0 image-1 image-2 image-3"; }
.image-layouts-layout-single { grid-template-columns: 1fr; grid-template-areas: "image-0"; }

/* ===== 图片单元格与定位 ===== */
.image-layouts-image-0 { grid-area: image-0; }
.image-layouts-image-1 { grid-area: image-1; }
.image-layouts-image-2 { grid-area: image-2; }
.image-layouts-image-3 { grid-area: image-3; }
.image-layouts-image-4 { grid-area: image-4; }
.image-layouts-image-5 { grid-area: image-5; }
.image-layouts-image-6 { grid-area: image-6; }
/* 超出 6 张的（custom 最多 20）用内联 grid-area，无需 class */

.image-layouts-image-cell { position: relative; overflow: hidden; min-width: 0; }
.image-layouts-image-cell img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
  object-position: center;
}
/* fit 规则同时支持「容器级」（方案 A 输出）与「单元格级」（方案 B 输出） */
.image-layouts-image-cell.image-layouts-fit-contain img,
.image-layouts-grid.image-layouts-fit-contain img { object-fit: contain; }
.image-layouts-image-cell.image-layouts-fit-natural img,
.image-layouts-grid.image-layouts-fit-natural img { height: auto; object-fit: unset; }

/* ===== Overlay（描述条） ===== */
.image-layouts-overlay {
  position: absolute;
  bottom: 0; left: 0; right: 0;
  display: flex; align-items: flex-end;
  padding: 1rem;
  pointer-events: none;
  transition: opacity 0.15s ease;
}
.image-layouts-overlay-hidden { opacity: 0; }
.image-layouts-image-cell:hover .image-layouts-overlay-hidden { opacity: 1; }
.image-layouts-overlay-text {
  width: 100%;
  border-radius: 0.375rem;
  background: rgba(255, 255, 255, 0.75);
  -webkit-backdrop-filter: blur(4px);
  backdrop-filter: blur(4px);
  padding: 0.5rem 1rem;
  text-align: center;
  font-size: 0.875rem;
  font-weight: 500;
  color: #111827;
}

/* ===== Caption ===== */
.image-layouts-caption {
  text-align: center;
  font-size: 0.75rem;
  margin: 0.5rem 0;
  color: #6b7280; /* 站点弱化色；Obsidian 中为 var(--text-muted) */
}

/* ===== Align / Width ===== */
.image-layouts-align { display: flex; }
.image-layouts-align-left { justify-content: flex-start; }
.image-layouts-align-center { justify-content: center; }
.image-layouts-align-right { justify-content: flex-end; }

/* ===== Masonry ===== */
.image-layouts-masonry-grid-2 { display: grid; grid-template-columns: repeat(2, 1fr); grid-gap: 0.5rem; }
.image-layouts-masonry-grid-3 { display: grid; grid-template-columns: repeat(3, 1fr); grid-gap: 0.5rem; }
.image-layouts-masonry-grid-4 { display: grid; grid-template-columns: repeat(4, 1fr); grid-gap: 0.5rem; }
.image-layouts-masonry-grid-5 { display: grid; grid-template-columns: repeat(5, 1fr); grid-gap: 0.5rem; }
.image-layouts-masonry-grid-6 { display: grid; grid-template-columns: repeat(6, 1fr); grid-gap: 0.5rem; }
.image-layouts-masonry-column { display: flex; flex-direction: column; gap: 0.5rem; }

/* ===== Carousel ===== */
.image-layout-carousel {
  position: relative;
  border-radius: 0.5rem;
  padding: 0.5rem 2rem;
}
.slides-container {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border-radius: 0.5rem;
}
.slides-container .slide { display: none; }
.slides-container .slide.active { display: block; }
.slides-container .slide img { height: 100%; width: auto; max-width: 100%; object-fit: contain; object-position: center; }
.slide-caption {
  position: absolute;
  bottom: 0; left: 0; right: 0;
  padding: 0.5rem 1rem;
  text-align: center;
  font-size: 0.875rem;
  color: #6b7280;
}
.nav-button {
  position: absolute;
  top: 50%; transform: translateY(-50%);
  z-index: 10;
  width: 2.5rem; height: 2.5rem;
  border-radius: 9999px;
  border: none;
  background: rgba(0, 0, 0, 0.35);
  color: #fff;
  font-size: 1.5rem;
  line-height: 1;
  cursor: pointer;
}
.nav-button.prev { left: 0.25rem; }
.nav-button.next { right: 0.25rem; }

/* 缩略图条（carouselShowThumbnails: true 时由 JS 注入） */
.carousel-thumbnails {
  display: flex;
  gap: 0.5rem;
  overflow-x: auto;
  margin-top: 0.5rem;
  padding-bottom: 0.25rem;
}
.carousel-thumbnails img {
  width: 6rem; height: 4rem;
  object-fit: cover;
  border-radius: 0.375rem;
  cursor: pointer;
  opacity: 0.6;
  border: 2px solid transparent;
}
.carousel-thumbnails img.active { opacity: 1; border-color: #4f46e5; }

/* 圆点按钮（默认形态） */
.carousel-pills {
  display: flex;
  justify-content: center;
  gap: 0.75rem;
  margin-top: 0.5rem;
}
.carousel-pills button {
  width: 0.625rem; height: 0.625rem;
  border-radius: 9999px;
  border: none;
  background: #d1d5db;
  cursor: pointer;
  padding: 0;
}
.carousel-pills button.active { background: #4f46e5; }

/* 错误提示（自定义网格非法时） */
.image-layouts-error {
  color: #888;
  font-size: 0.85em;
  padding: 0.5rem;
  border: 1px dashed #8886;
  border-radius: 6px;
  white-space: pre-wrap;
}
```

### 9.2 Carousel 客户端 JS（可选增强，保存为 `static/js/image-layouts-carousel.js`）

> 仅当页面包含 carousel 时引入。静态 HTML 已显示第一张（无 JS 也可读），JS 只负责交互。

```js
document.querySelectorAll(".image-layout-carousel").forEach((root) => {
  const slides = Array.from(root.querySelectorAll(".slide"));
  if (slides.length === 0) return;
  const thumbs = root.dataset.thumbnails === "true";
  let current = 0;

  const pills = document.createElement("div");
  pills.className = "carousel-pills";
  const thumbStrip = document.createElement("div");
  thumbStrip.className = "carousel-thumbnails";

  slides.forEach((_, i) => {
    if (thumbs) {
      const img = document.createElement("img");
      img.src = slides[i].querySelector("img").src;
      img.alt = `Thumbnail ${i + 1}`;
      img.addEventListener("click", () => go(i));
      thumbStrip.appendChild(img);
    } else {
      const b = document.createElement("button");
      b.type = "button";
      b.setAttribute("aria-label", `第 ${i + 1} 张`);
      b.addEventListener("click", () => go(i));
      pills.appendChild(b);
    }
  });

  if (thumbs) root.appendChild(thumbStrip); else root.appendChild(pills);

  function go(i) {
    slides[current].classList.remove("active");
    if (thumbs) thumbStrip.children[current]?.classList.remove("active");
    else pills.children[current]?.classList.remove("active");
    current = (i + slides.length) % slides.length;
    slides[current].classList.add("active");
    if (thumbs) thumbStrip.children[current]?.classList.add("active");
    else pills.children[current]?.classList.add("active");
    if (thumbs) thumbStrip.children[current]?.scrollIntoView({ behavior: "smooth", block: "nearest", inline: "center" });
  }

  root.querySelector(".nav-button.prev").addEventListener("click", () => go(current - 1));
  root.querySelector(".nav-button.next").addEventListener("click", () => go(current + 1));
  root.tabIndex = 0;
  root.addEventListener("keydown", (e) => {
    if (e.key === "ArrowLeft") go(current - 1);
    if (e.key === "ArrowRight") go(current + 1);
  });

  if (thumbs) thumbStrip.children[0]?.classList.add("active");
  else pills.children[0]?.classList.add("active");
});
```

---

## 10. 实施步骤清单

> Agent 按序执行。每步完成后在验收清单（§11）打勾。

1. **确认项目类型**：标准 Hugo（≥ 0.127）→ 方案 A；自研/其它 → 方案 B。
2. **复制共享资产**：`image-layouts.css` → 站点 `static/css/`；需要 carousel 时复制 `image-layouts-carousel.js` → `static/js/`，并在模板 `<head>` / 页脚引入。
3. **方案 A 步骤**：
   a. 创建 `layouts/partials/image-layouts/render.html`（§7.4 完整代码）；
   b. 创建 modern 薄文件 + 用 §7.5 命令生成 17 个 legacy 薄文件；
   c. 按需调整 `$IMG_ROOT`（§7.3 路径映射）；
   d. 检查 `hugo version` ≥ 0.127，必要时升级；
   e. 处理 `fromFolder`：确认图片目录结构（`static/images/<folder>`）。
4. **方案 B 步骤**：
   a. 复制 `scripts/image-layouts.mjs`（§8.2）与插件纯逻辑文件（§5.1）；
   b. 用 `gray-matter` 替换最小 YAML 解析器（可选但建议）；
   c. 接入构建流程（§8.3）；配置 `imageRoot`；
   d. 若目标 SSG 需要 raw HTML，打开对应配置（Hugo：`unsafe = true`）。
5. **图片资产迁移**：把 Obsidian 仓库中的图片按相对路径复制到站点 `static/images/`；远程图片无需处理。
6. **用 test-vault 示例验收**：把本仓库 `test-vault/` 下 3 个 md 的内容（或 §3.6 示例）放入一篇测试文章，构建并核对 §11。
7. **边界处理**：空的 `image-layout` 块（无 layout）→ 输出注释或空容器，不报错；非法 custom grid → 输出 `.image-layouts-error` 提示，不静默渲染。
8. **清理**：删除临时转换文件；确认普通代码块（```js 等）渲染不受影响。

---

## 11. 验收测试清单

| # | 用例 | 期望 |
| --- | --- | --- |
| 1 | `image-layout-a` 两块图 | 左右各 50%，间距 0.5rem，图片 cover 铺满 |
| 2 | `image-layout-c` | **左小右大**（`image-1` 在左）；图片 0 渲染在右侧大格 |
| 3 | `image-layout-d`（3 张） | 左大图占整列两行；右上/右下各一图 |
| 4 | `image-layout-single` + `align: right` | 布局靠右，宽度 50%（或 `width` 值） |
| 5 | `fit: contain` | 图片不裁切，留白边 |
| 6 | `fit: natural` | 图片保持自身比例，行内按高对齐（align-items: start） |
| 7 | wiki 链接 `![[img\|desc]]` | 渲染 `<img alt="desc">` + hover 显示描述条 |
| 8 | wiki 尺寸 `![[img\|300]]` | 不产生额外宽度约束（站点场景可忽略该特性，但不得把 "300" 当描述） |
| 9 | `overlay: never` | 无 overlay 元素 |
| 10 | `overlay: always` / `permanentOverlay: true` | 描述条常显 |
| 11 | `descriptions` 数组 | 覆盖图片行自带描述 |
| 12 | `caption` | 布局下方居中显示小字 |
| 13 | `image-layout-masonry-3` 9 张 | 3 列，第 0、3、6 张在第一列（index % 3） |
| 14 | `fromFolder` + `sortBy: mtime` + `reverse` + `limit` | 文件夹图片按规则追加在块内图片后 |
| 15 | 图片不足 | 占位图填充至布局所需数 |
| 16 | 图片过多 | 截断到所需数 |
| 17 | `layout: custom`（A A B / A A C） | 左 2/3 高图 + 右上下两图 |
| 18 | custom 非法网格（非矩形、行长不一、空） | 显示错误提示，不渲染错乱 |
| 19 | `layout: carousel` | 显示第一张 + 圆点；点圆点/箭头/键盘可切换并循环 |
| 20 | `carouselShowThumbnails: true` | 缩略图条，点击跳转，当前项高亮 |
| 21 | modern `layout: legacy-layout-a` 写法 | 等价 `a` |
| 22 | 普通代码块（```js） | 仍正常渲染，不受影响（方案 A 关键回归项） |
| 23 | 空 `image-layout` 块 | 无崩溃，输出注释/空容器 |
| 24 | 远程图片 URL | 原样 `<img src>` |
| 25 | `~~~image-layout-a` 波浪线 fence | 同样生效 |

---

## 12. 附录：插件源码文件 → 迁移策略映射

| 插件文件 | 性质 | 迁移处理 |
| --- | --- | --- |
| `src/utils/images.ts` | 纯逻辑 | 复制（方案 B 已内置） |
| `src/utils/custom-grid.ts` | 纯逻辑 | 复制（方案 B 已内置；方案 A 需在模板中实现/简化） |
| `src/utils/front-matter.ts` | 纯逻辑（依赖 npm `front-matter`） | 复制或换 `gray-matter` |
| `src/utils/options.ts` / `overlay.ts` / `placeholder.ts` | 纯逻辑 | 复制（去掉 Obsidian 设置部分） |
| `src/interfaces.ts` | 常量映射 | 复制（§3.4 数据源） |
| `src/utils/blocks.ts` | 纯逻辑 + 编辑回写 | 仅复制 `parseMasonryLayoutName` / fence 工具 |
| `src/utils/image-resolver.ts` / `folder-images.ts` | Obsidian API | 替换为路径映射 / fs 扫描（§7.3、§8.2） |
| `src/processors/*.ts` | Obsidian 注册机制 | 替换为模板 partial / 预处理函数 |
| `src/components/*.svelte` | Svelte + CSS-in-JS | 重写为静态 HTML + §9.1 CSS |
| `src/views/*`、`SwitchableLayout`、`LayoutPicker`、`editor-writeback` | 编辑态 UI | 剥离 |
| `test-vault/publish.ts` | Obsidian Publish 静态渲染参考 | **最接近迁移目标的参考实现**，方案 B 的直接蓝本 |
| `docs/options.md` 等 | 官方文档 | 语法规范备份（本指南已汇总） |

---

*文档结束。生成依据：obsidian-image-layouts v0.18.0 源码（src/、test-vault/publish.ts）与官方文档（docs/）。*
