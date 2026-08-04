// Package imagelayout 内置 image-layout 图片结构化渲染组件：把 ```image-layout* 围栏代码块
// 渲染为静态图片网格 / 瀑布流 / 轮播 / 自定义网格，语法规范见 docs/MIGRATION-GUIDE.md。
package imagelayout

import (
	_ "embed"
	"errors"
	"fmt"
	"html"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"rainhush/internal/markdown/components"

	"gopkg.in/yaml.v3"
)

//go:embed assets/image-layouts.css
var cssAsset []byte

//go:embed assets/image-layouts-carousel.js
var carouselJSAsset []byte

//go:embed assets/image-layouts-loader.js
var loaderJSAsset []byte

type component struct{}

var _ components.Component = (*component)(nil)

func init() { components.Register(&component{}) }

func (c *component) Name() string { return "image-layout" }

func (c *component) CSS() [][]byte { return [][]byte{cssAsset} }
func (c *component) JS() [][]byte {
	return [][]byte{carouselJSAsset, loaderJSAsset}
}

const (
	layoutPrefix   = "image-layout"
	imageRootURL   = "/images/"
	imageRootDir   = "static/images/"
	maxCustomSlots = 20
)

var placeholderStyle = `data:image/svg+xml,` + encodeDataURI(`<svg xmlns="http://www.w3.org/2000/svg" width="640" height="480"><rect width="100%" height="100%" fill="#88888822"/><circle cx="240" cy="170" r="36" fill="#88888855"/><path d="M120 360l110-140 80 95 60-60 130 105z" fill="#88888855"/></svg>`)

var (
	mdImgRe   = regexp.MustCompile(`!\[([^\]]*)\]\(([^()]*(?:\([^()]*\)[^()]*)*)\)`)
	wikiRe    = regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	sizeSegRe = regexp.MustCompile(`^\d+(?:x\d+)?$`)
	imageExts = map[string]bool{
		"avif": true, "bmp": true, "gif": true, "jpeg": true,
		"jpg": true, "png": true, "svg": true, "webp": true,
	}
)

type image struct {
	link     string
	alt      string
	external bool
	width    int
	height   int
}

type options struct {
	caption        string
	descriptions   []string
	overlay        string
	fit            string
	align          string
	width          string
	fromFolder     string
	sortBy         string
	reverse        bool
	limit          int
	grid           string
	carouselThumbs bool
	carouselBg     string
	carouselHeight string
}

type gridTemplate struct {
	columns string
	areas   string
	slots   int
}

var gridTemplates = map[string]gridTemplate{
	"a":      {"1fr 1fr", `"image-0 image-1"`, 2},
	"b":      {"2fr 1fr", `"image-0 image-1"`, 2},
	"c":      {"1fr 2fr", `"image-1 image-0"`, 2},
	"d":      {"2fr 1fr", `"image-0 image-1" "image-0 image-2"`, 3},
	"e":      {"1fr 2fr", `"image-1 image-0" "image-2 image-0"`, 3},
	"f":      {"3fr 1fr", `"image-0 image-1" "image-0 image-2" "image-0 image-3"`, 4},
	"g":      {"1fr 3fr", `"image-1 image-0" "image-2 image-0" "image-3 image-0"`, 4},
	"h":      {"1fr 1fr 1fr", `"image-0 image-1 image-2"`, 3},
	"i":      {"1fr 1fr 1fr 1fr", `"image-0 image-1 image-2 image-3"`, 4},
	"single": {"1fr", `"image-0"`, 1},
}

type customGrid struct {
	columns int
	slots   int
	areas   string
}

// Render 处理 image-layout 系列围栏代码块；doc 是当前渲染文档路径（仅用于日志）。
// 返回 ok=true 表示已接管渲染，false 表示非本语法，应交由默认渲染器。
func (c *component) Render(lang string, content string, doc string) (string, bool) {
	suffix, ok := parseFenceLang(lang)
	if !ok {
		return "", false
	}

	data, body := parseFrontMatter(content)

	layout, defaultAlign := "", "full"
	if suffix != "" {
		layout = suffix
		switch suffix {
		case "left":
			layout, defaultAlign = "single", "left"
		case "center":
			layout, defaultAlign = "single", "center"
		case "right":
			layout, defaultAlign = "single", "right"
		}
	} else {
		layout = str(data, "layout")
	}
	layout = normalizeLayout(layout)

	if layout == "" {
		log.Printf("[image-layout] %s: empty block (lang=%q), rendered as placeholder comment", doc, lang)
		return "<!-- image-layout: empty block (no layout specified) -->", true
	}

	o := normalizeOptions(data, defaultAlign)
	imgs := collectImages(body)

	if o.fromFolder != "" {
		folder := folderImages(o.fromFolder, o)
		log.Printf("[image-layout] %s: fromFolder %q -> %d images (sort=%s reverse=%v limit=%d)", doc, o.fromFolder, len(folder), o.sortBy, o.reverse, o.limit)
		imgs = append(imgs, folder...)
	}

	resolveDimensions(imgs)

	log.Printf("[image-layout] %s: lang=%q layout=%q images=%d overlay=%s fit=%s align=%s", doc, lang, layout, len(imgs), o.overlay, o.fit, o.align)
	for i, img := range imgs {
		dim := ""
		if img.width > 0 {
			dim = fmt.Sprintf(" %dx%d", img.width, img.height)
		}
		log.Printf("[image-layout] %s:   image[%d] src=%q alt=%q dim=%s", doc, i, img.link, img.alt, dim)
	}

	out := render(layout, imgs, o)
	log.Printf("[image-layout] %s: rendered %q -> %d bytes", doc, layout, len(out))
	return out, true
}

// resolveDimensions 为本地图片探测固有尺寸（远程图与探测失败保持 0），
// 渲染时输出 width/height 属性，让浏览器加载前即按比例占位，消除布局抖动。
func resolveDimensions(imgs []image) {
	for i := range imgs {
		if imgs[i].external {
			continue
		}
		if w, h, ok := imageDimensions(imgs[i].link); ok {
			imgs[i].width, imgs[i].height = w, h
		}
	}
}

func parseFenceLang(lang string) (suffix string, ok bool) {
	if lang == layoutPrefix {
		return "", true
	}
	if strings.HasPrefix(lang, layoutPrefix+"-") {
		return strings.TrimPrefix(lang, layoutPrefix+"-"), true
	}
	return "", false
}

func normalizeLayout(l string) string {
	switch {
	case strings.HasPrefix(l, "legacy-layout-"):
		return strings.TrimPrefix(l, "legacy-layout-")
	case strings.HasPrefix(l, "legacy-masonry-"):
		return "masonry-" + strings.TrimPrefix(l, "legacy-masonry-")
	default:
		return l
	}
}

// parseFrontMatter 分割块内 YAML front matter 与正文；YAML 解析失败时 data 为 nil，正文按原文处理。
func parseFrontMatter(content string) (map[string]interface{}, string) {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil, content
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "---" {
			continue
		}
		var data map[string]interface{}
		if err := yaml.Unmarshal([]byte(strings.Join(lines[1:i], "\n")), &data); err != nil {
			log.Printf("[image-layout] front matter parse failed (falling back to plain content): %v", err)
			return nil, content
		}
		return data, strings.Join(lines[i+1:], "\n")
	}
	return nil, content
}

func collectImages(body string) []image {
	var imgs []image
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "!") {
			continue
		}
		if img := parseImageLine(trimmed); img != nil {
			if !img.external {
				img.link = imageRootURL + img.link
			}
			imgs = append(imgs, *img)
		}
	}
	return imgs
}

func parseImageLine(line string) *image {
	if !strings.HasPrefix(line, "!") {
		return nil
	}
	if m := mdImgRe.FindStringSubmatch(line); m != nil {
		alt, _ := splitSegs(strings.Split(m[1], "|"))
		link := m[2]
		return &image{link: link, alt: alt, external: isExternal(link)}
	}
	if m := wikiRe.FindStringSubmatch(line); m != nil {
		segs := strings.Split(m[1], "|")
		link := decodePath(strings.TrimSpace(segs[0]))
		if link == "" {
			return nil
		}
		alt, _ := splitSegs(segs[1:])
		return &image{link: link, alt: alt}
	}
	return nil
}

func splitSegs(segs []string) (alt string, _ int) {
	var altParts []string
	for _, s := range segs {
		t := strings.TrimSpace(s)
		if t == "" || sizeSegRe.MatchString(t) {
			continue
		}
		altParts = append(altParts, t)
	}
	return strings.Join(altParts, "|"), 0
}

func decodePath(s string) string {
	if !strings.Contains(s, "%") {
		return s
	}
	if dec, err := url.PathUnescape(s); err == nil {
		return dec
	}
	return s
}

func isExternal(link string) bool {
	return strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://")
}

func normalizeOptions(data map[string]interface{}, defaultAlign string) options {
	o := options{
		caption:        str(data, "caption"),
		descriptions:   descriptions(data),
		overlay:        resolveOverlay(data),
		fit:            normalizeFit(str(data, "fit")),
		align:          normalizeAlign(str(data, "align"), defaultAlign),
		width:          str(data, "width"),
		fromFolder:     str(data, "fromFolder"),
		sortBy:         str(data, "sortBy"),
		reverse:        boolV(data, "reverse"),
		limit:          intV(data, "limit"),
		grid:           str(data, "grid"),
		carouselThumbs: boolV(data, "carouselShowThumbnails"),
		carouselBg:     str(data, "carouselBackground"),
		carouselHeight: str(data, "carouselHeight"),
	}
	if o.width == "" {
		o.width = "50%"
	} else if isNumeric(o.width) {
		o.width += "px"
	}
	if o.sortBy == "" {
		o.sortBy = "name"
	}
	return o
}

func resolveOverlay(data map[string]interface{}) string {
	switch v := str(data, "overlay"); v {
	case "never", "hover", "always":
		return v
	}
	if permanentOverlay, ok := data["permanentOverlay"].(bool); ok {
		if permanentOverlay {
			return "always"
		}
		return "hover"
	}
	return "hover"
}

func normalizeFit(v string) string {
	switch v {
	case "cover", "contain", "natural":
		return v
	}
	return "cover"
}

func normalizeAlign(v, fallback string) string {
	switch v {
	case "left", "center", "right", "full":
		return v
	}
	return fallback
}

func descriptions(data map[string]interface{}) []string {
	raw, ok := data["descriptions"].([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, len(raw))
	for i, v := range raw {
		if s, ok := v.(string); ok {
			out[i] = s
		}
	}
	return out
}

func str(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func boolV(m map[string]interface{}, key string) bool {
	v, _ := m[key].(bool)
	return v
}

func intV(m map[string]interface{}, key string) int {
	switch v := m[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return 0
}

func folderImages(folder string, o options) []image {
	dir := filepath.Join(imageRootDir, filepath.FromSlash(folder))
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("[image-layout] fromFolder %q: %v", folder, err)
		return nil
	}
	type fileInfo struct {
		name  string
		mtime time.Time
	}
	var files []fileInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !imageExts[strings.ToLower(strings.TrimPrefix(filepath.Ext(e.Name()), "."))] {
			continue
		}
		if info, err := e.Info(); err == nil {
			files = append(files, fileInfo{e.Name(), info.ModTime()})
		}
	}
	sort.Slice(files, func(i, j int) bool {
		if o.sortBy == "mtime" {
			return files[i].mtime.Before(files[j].mtime)
		}
		return files[i].name < files[j].name
	})
	if o.reverse {
		for i, j := 0, len(files)-1; i < j; i, j = i+1, j-1 {
			files[i], files[j] = files[j], files[i]
		}
	}
	if o.limit > 0 && len(files) > o.limit {
		files = files[:o.limit]
	}
	imgs := make([]image, 0, len(files))
	for _, f := range files {
		imgs = append(imgs, image{link: path.Join(imageRootURL, folder, f.name)})
	}
	return imgs
}

func render(layout string, imgs []image, o options) string {
	switch {
	case layout == "carousel":
		return withCaption(renderCarousel(imgs, o), o.caption)
	case layout == "custom":
		g, err := parseCustomGrid(o.grid)
		if err != nil {
			log.Printf("[image-layout] %s: custom grid error: %v", layout, err)
			return `<p class="image-layouts-error">Image Layouts: ` + html.EscapeString(err.Error()) + `</p>`
		}
		return withCaption(renderCustom(g, imgs, o), o.caption)
	case strings.HasPrefix(layout, "masonry-"):
		n, err := strconv.Atoi(strings.TrimPrefix(layout, "masonry-"))
		if err != nil || n < 2 || n > 6 {
			return unknownLayout(layout)
		}
		return withCaption(renderMasonry(imgs, n, o), o.caption)
	default:
		t, ok := gridTemplates[layout]
		if !ok {
			return unknownLayout(layout)
		}
		return withCaption(renderGrid(t, layout, imgs, o), o.caption)
	}
}

func unknownLayout(layout string) string {
	log.Printf("[image-layout] unknown layout %q", layout)
	return `<p class="image-layouts-error">Image Layouts: unknown layout "` + html.EscapeString(layout) + `"</p>`
}

func renderCarousel(imgs []image, o options) string {
	height := o.carouselHeight
	if height == "" {
		height = "24rem"
	} else if isNumeric(height) {
		height += "px"
	}
	bg := o.carouselBg
	if bg == "" {
		bg = "#f1f5f9"
	}
	thumbs := "false"
	if o.carouselThumbs {
		thumbs = "true"
	}

	var b strings.Builder
	b.WriteString(`<div class="image-layout-carousel" data-thumbnails="` + thumbs + `" style="background:` + html.EscapeString(bg) + `"><div class="slides-container" style="height:` + html.EscapeString(height) + `">`)
	for i, img := range imgs {
		d := desc(i, img, o)
		b.WriteString(`<div class="slide`)
		if i == 0 {
			b.WriteString(` active`)
		}
		b.WriteString(`"><img src="` + html.EscapeString(img.link) + `" alt="` + html.EscapeString(altText(d, i)) + `" ` + imgAttrs(img, i) + `>`)
		if d != "" {
			b.WriteString(`<div class="slide-caption">` + html.EscapeString(d) + `</div>`)
		}
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div><button class="nav-button prev" type="button" aria-label="上一张">‹</button><button class="nav-button next" type="button" aria-label="下一张">›</button></div>`)
	return b.String()
}

func renderCustom(g customGrid, imgs []image, o options) string {
	imgs = padToSlots(imgs, g.slots)
	var b strings.Builder
	b.WriteString(`<div class="image-layouts image-layouts-grid image-layouts-custom" style="grid-template-columns: repeat(` + strconv.Itoa(g.columns) + `, 1fr); grid-template-areas: ` + html.EscapeString(g.areas) + `;">`)
	for i := 0; i < g.slots; i++ {
		b.WriteString(renderCell(imgs[i], i, o, "", fmt.Sprintf("image-%d", i)))
	}
	b.WriteString(`</div>`)
	return b.String()
}

func renderMasonry(imgs []image, cols int, o options) string {
	var b strings.Builder
	b.WriteString(`<div class="image-layouts-masonry-grid-` + strconv.Itoa(cols) + `">`)
	for c := 0; c < cols; c++ {
		b.WriteString(`<div class="image-layouts-masonry-column">`)
		for i := c; i < len(imgs); i += cols {
			b.WriteString(renderCell(imgs[i], i, o, "", ""))
		}
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

func renderGrid(t gridTemplate, layout string, imgs []image, o options) string {
	imgs = padToSlots(imgs, t.slots)
	var b strings.Builder
	b.WriteString(`<div class="image-layouts-align image-layouts-align-` + html.EscapeString(o.align) + `"><div class="image-layouts image-layouts-grid image-layouts-layout-` + html.EscapeString(layout) + ` image-layouts-fit-` + html.EscapeString(o.fit) + `" style="grid-template-columns: ` + html.EscapeString(t.columns) + `; grid-template-areas: ` + html.EscapeString(t.areas) + `;`)
	if o.align != "full" {
		b.WriteString(` width:` + html.EscapeString(o.width) + `;max-width:` + html.EscapeString(o.width) + `;`)
	}
	b.WriteString(`">`)
	for i := 0; i < t.slots; i++ {
		b.WriteString(renderCell(imgs[i], i, o, fmt.Sprintf("image-layouts-image-%d", i), ""))
	}
	b.WriteString(`</div></div>`)
	return b.String()
}

func renderCell(img image, i int, o options, cellClass, gridArea string) string {
	d := desc(i, img, o)
	var b strings.Builder
	b.WriteString(`<div class="image-layouts-image-cell`)
	if cellClass != "" {
		b.WriteString(` ` + cellClass)
	}
	if o.fit == "contain" {
		b.WriteString(` image-layouts-fit-contain`)
	}
	if o.fit == "natural" {
		b.WriteString(` image-layouts-fit-natural`)
	}
	b.WriteString(`"`)
	if gridArea != "" {
		b.WriteString(` style="grid-area: ` + html.EscapeString(gridArea) + `"`)
	}
	b.WriteString(`><img ` + srcAttr(img) + `="` + html.EscapeString(img.link) + `" alt="` + html.EscapeString(altText(d, i)) + `" ` + imgAttrs(img, i) + `>`)
	if d != "" && o.overlay != "never" {
		b.WriteString(`<div class="image-layouts-overlay`)
		if o.overlay != "always" {
			b.WriteString(` image-layouts-overlay-hidden`)
		}
		b.WriteString(`"><div class="image-layouts-overlay-text">` + html.EscapeString(d) + `</div></div>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

func desc(i int, img image, o options) string {
	if i < len(o.descriptions) && o.descriptions[i] != "" {
		return o.descriptions[i]
	}
	return img.alt
}

// imgAttrs 构建 <img> 加载属性：本地图首张 eager + 高优先级立即加载，其余懒加载；
// 远程图一律懒加载（实际加载时机由 image-layouts-loader.js 通过 data-src 控制，
// 此处 loading/decoding 仅作降级兜底）。已知尺寸的本地图输出 width/height，
// 浏览器加载前即按比例占位，避免布局抖动。
func imgAttrs(img image, index int) string {
	attrs := `decoding="async"`
	if !img.external && index == 0 {
		attrs += ` fetchpriority="high" loading="eager"`
	} else {
		attrs += ` loading="lazy"`
	}
	if img.width > 0 && img.height > 0 {
		attrs += fmt.Sprintf(` width="%d" height="%d"`, img.width, img.height)
	}
	return attrs
}

// srcAttr 返回图片地址属性名：远程图用 data-src（由 loader.js 在进入视口时
// 才赋值给 src，避免 innerHTML 注入瞬间浏览器并发拉取/解码，与页面其它功能解耦）；
// 本地图直接用 src（同源快、有固有尺寸、无 JS 也能显示）。
func srcAttr(img image) string {
	if img.external {
		return "data-src"
	}
	return "src"
}

func altText(d string, i int) string {
	if d != "" {
		return d
	}
	return fmt.Sprintf("Image %d", i+1)
}

func withCaption(htmlContent, caption string) string {
	if caption == "" {
		return htmlContent
	}
	return htmlContent + `<div class="image-layouts-caption">` + html.EscapeString(caption) + `</div>`
}

func padToSlots(imgs []image, slots int) []image {
	if len(imgs) >= slots {
		return imgs[:slots]
	}
	out := make([]image, len(imgs), slots)
	copy(out, imgs)
	for len(out) < slots {
		out = append(out, image{link: placeholderStyle})
	}
	return out
}

func parseCustomGrid(spec string) (customGrid, error) {
	if strings.TrimSpace(spec) == "" {
		return customGrid{}, errors.New("A custom layout needs a `grid` option with rows of letters, e.g.\ngrid: |\n  A A B\n  A A C")
	}
	var rows [][]string
	for _, line := range strings.Split(spec, "\n") {
		if t := strings.TrimSpace(line); t != "" {
			rows = append(rows, strings.Fields(t))
		}
	}
	if len(rows) == 0 {
		return customGrid{}, errors.New("A custom layout needs a `grid` option with rows of letters")
	}
	cols := len(rows[0])
	for _, r := range rows {
		if len(r) != cols {
			return customGrid{}, errors.New("Every row in `grid` must have the same number of cells.")
		}
	}
	var order []string
	seen := make(map[string]bool)
	for _, r := range rows {
		for _, cell := range r {
			if cell != "." && !seen[cell] {
				seen[cell] = true
				order = append(order, cell)
			}
		}
	}
	if len(order) == 0 {
		return customGrid{}, errors.New("The `grid` needs at least one image cell.")
	}
	if len(order) > maxCustomSlots {
		return customGrid{}, fmt.Errorf("`grid` supports up to %d images.", maxCustomSlots)
	}
	for _, token := range order {
		minR, maxR, minC, maxC, count := len(rows), -1, cols, -1, 0
		for r := range rows {
			for c := range rows[r] {
				if rows[r][c] != token {
					continue
				}
				if r < minR {
					minR = r
				}
				if r > maxR {
					maxR = r
				}
				if c < minC {
					minC = c
				}
				if c > maxC {
					maxC = c
				}
				count++
			}
		}
		if count != (maxR-minR+1)*(maxC-minC+1) {
			return customGrid{}, fmt.Errorf("The cells for %q must form a solid rectangle.", token)
		}
	}
	var areaRows []string
	for _, r := range rows {
		cells := make([]string, len(r))
		for c, cell := range r {
			if cell == "." {
				cells[c] = "."
			} else {
				cells[c] = fmt.Sprintf("image-%d", indexOf(order, cell))
			}
		}
		areaRows = append(areaRows, `"`+strings.Join(cells, " ")+`"`)
	}
	return customGrid{columns: cols, slots: len(order), areas: strings.Join(areaRows, " ")}, nil
}

func indexOf(list []string, s string) int {
	for i, v := range list {
		if v == s {
			return i
		}
	}
	return -1
}

func isNumeric(s string) bool {
	_, err := strconv.Atoi(strings.TrimSpace(s))
	return err == nil
}

func encodeDataURI(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c == '%':
			b.WriteString("%25")
		case c == '"':
			b.WriteString("%22")
		case c == '#':
			b.WriteString("%23")
		case c == '<':
			b.WriteString("%3C")
		case c == '>':
			b.WriteString("%3E")
		case c == ' ':
			b.WriteString("%20")
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}
