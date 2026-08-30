package imagelayout

import (
	"strings"
	"testing"
)

func TestParseFenceLang(t *testing.T) {
	cases := []struct {
		lang   string
		suffix string
		ok     bool
	}{
		{"image-layout", "", true},
		{"image-layout-a", "a", true},
		{"image-layout-masonry-3", "masonry-3", true},
		{"image-layout-left", "left", true},
		{"image-layout-center", "center", true},
		{"image-layout-right", "right", true},
		{"image-layout-single", "single", true},
		{"image-layouts", "", false},
		{"mermaid", "", false},
		{"go", "", false},
	}
	for _, c := range cases {
		suffix, ok := parseFenceLang(c.lang)
		if ok != c.ok || suffix != c.suffix {
			t.Errorf("parseFenceLang(%q) = (%q, %v), want (%q, %v)", c.lang, suffix, ok, c.suffix, c.ok)
		}
	}
}

func TestParseImageLine(t *testing.T) {
	cases := []struct {
		line     string
		link     string
		alt      string
		external bool
	}{
		{"![[beach.jpg]]", "beach.jpg", "", false},
		{"![[beach.jpg|Low tide]]", "beach.jpg", "Low tide", false},
		{"![[beach.jpg|300]]", "beach.jpg", "", false},
		{"![[beach.jpg|300x200]]", "beach.jpg", "", false},
		{"![[beach.jpg|Low tide|300]]", "beach.jpg", "Low tide", false},
		{"![[beach.jpg|a|b|c]]", "beach.jpg", "a|b|c", false},
		{"![[Test%20folder/img.jpg]]", "Test folder/img.jpg", "", false},
		{"![Low tide](beach.jpg)", "beach.jpg", "Low tide", false},
		{"![x](https://example.com/x.jpg)", "https://example.com/x.jpg", "x", true},
		{"![Alt (1)](Screenshot (1).png)", "Screenshot (1).png", "Alt (1)", false},
		{"[[not-an-image]]", "", "", false},
		{"plain text", "", "", false},
	}
	for _, c := range cases {
		img := parseImageLine(c.line)
		if c.link == "" {
			if img != nil {
				t.Errorf("parseImageLine(%q) = %+v, want nil", c.line, img)
			}
			continue
		}
		if img == nil {
			t.Errorf("parseImageLine(%q) = nil, want link %q", c.line, c.link)
			continue
		}
		if img.link != c.link || img.alt != c.alt || img.external != c.external {
			t.Errorf("parseImageLine(%q) = %+v, want link=%q alt=%q external=%v", c.line, img, c.link, c.alt, c.external)
		}
	}
}

func TestParseCustomGrid(t *testing.T) {
	g, err := parseCustomGrid("A A B\nA A C")
	if err != nil {
		t.Fatalf("valid grid rejected: %v", err)
	}
	if g.columns != 3 || g.slots != 3 {
		t.Errorf("grid = %+v, want columns=3 slots=3", g)
	}
	want := `"image-0 image-0 image-1" "image-0 image-0 image-2"`
	if g.areas != want {
		t.Errorf("areas = %q, want %q", g.areas, want)
	}

	for _, spec := range []string{"A A\nB C", "A B\nA C", "A A\nA A"} {
		if _, err := parseCustomGrid(spec); err != nil {
			t.Errorf("valid rectangle grid %q rejected: %v", spec, err)
		}
	}

	invalid := []struct {
		spec string
		msg  string
	}{
		{"", "needs a `grid`"},
		{"A A\nA", "same number of cells"},
		{". .\n. .", "at least one image cell"},
		{"A A\nA B", "solid rectangle"}, // A 为 L 形，非实心矩形
		{"A B\nB A", "solid rectangle"},
		{"A B\nC A", "solid rectangle"},
	}
	for _, c := range invalid {
		if _, err := parseCustomGrid(c.spec); err == nil {
			t.Errorf("invalid grid %q accepted (want: %s)", c.spec, c.msg)
		}
	}
}

func TestParseFrontMatter(t *testing.T) {
	content := "---\nlayout: d\ncaption: hello\noverlay: always\ndescriptions:\n  - one\n  - two\nlimit: 5\nreverse: true\n---\n![[a.jpg]]\n![[b.jpg|desc]]\n"
	data, body := parseFrontMatter(content)
	if data == nil {
		t.Fatal("data is nil")
	}
	if data["layout"] != "d" || data["caption"] != "hello" || data["overlay"] != "always" {
		t.Errorf("unexpected data: %#v", data)
	}
	if !strings.Contains(body, "![[a.jpg]]") || strings.Contains(body, "---") {
		t.Errorf("unexpected body: %q", body)
	}
	if data["reverse"] != true || data["limit"] != 5 {
		t.Errorf("bool/int mismatch: %#v", data)
	}
	if _, ok := data["descriptions"].([]interface{}); !ok {
		t.Errorf("descriptions not a list: %#v", data["descriptions"])
	}

	// 无 front matter
	if data, body := parseFrontMatter("![[a.jpg]]\n"); data != nil || body == "" {
		t.Errorf("plain content mishandled: data=%v body=%q", data, body)
	}

	// 非法 YAML 回退为原文
	bad := "---\nlayout: [unclosed\n---\n![[a.jpg]]\n"
	if data, body := parseFrontMatter(bad); data != nil || !strings.Contains(body, "![[a.jpg]]") {
		t.Errorf("broken yaml not recovered: data=%v body=%q", data, body)
	}
}

func TestResolveOverlay(t *testing.T) {
	if v := resolveOverlay(map[string]interface{}{}); v != "hover" {
		t.Errorf("default = %q, want hover", v)
	}
	if v := resolveOverlay(map[string]interface{}{"overlay": "never"}); v != "never" {
		t.Errorf("overlay never = %q", v)
	}
	if v := resolveOverlay(map[string]interface{}{"permanentOverlay": true}); v != "always" {
		t.Errorf("permanentOverlay true = %q", v)
	}
	if v := resolveOverlay(map[string]interface{}{"overlay": "hover", "permanentOverlay": true}); v != "hover" {
		t.Errorf("overlay should win over permanentOverlay, got %q", v)
	}
}

var c = &component{}

func TestRenderFence(t *testing.T) {
	// legacy 预置网格
	out, ok := c.Render("image-layout-a", "![[a.jpg]]\n![[b.jpg|second]]\n", "test.md")
	if !ok {
		t.Fatalf("render a: not handled")
	}
	for _, want := range []string{"image-layouts-layout-a", "image-layouts-image-0", "image-layouts-image-1", "/images/a.jpg", "/images/b.jpg", "second"} {
		if !strings.Contains(out, want) {
			t.Errorf("render a missing %q in:\n%s", want, out)
		}
	}

	// modern + 图片不足 → 占位图
	out, ok = c.Render("image-layout", "---\nlayout: d\ncaption: cap\ndescriptions:\n  - one\n---\n![[a.jpg]]\n", "test.md")
	if !ok {
		t.Fatal("modern block not handled")
	}
	for _, want := range []string{"image-layouts-layout-d", "data:image/svg+xml", "one", "image-layouts-caption", "cap", "image-2"} {
		if !strings.Contains(out, want) {
			t.Errorf("render d missing %q in:\n%s", want, out)
		}
	}

	// masonry
	out, _ = c.Render("image-layout-masonry-3", "![[a.jpg]]\n![[b.jpg]]\n![[c.jpg]]\n![[d.jpg]]\n", "test.md")
	for _, want := range []string{"image-layouts-masonry-grid-3", "image-layouts-masonry-column"} {
		if !strings.Contains(out, want) {
			t.Errorf("render masonry missing %q", want)
		}
	}

	// carousel
	out, _ = c.Render("image-layout", "---\nlayout: carousel\ncarouselShowThumbnails: true\n---\n![[sunset.jpg|Sunset]]\n", "test.md")
	for _, want := range []string{"image-layout-carousel", "data-thumbnails=\"true\"", "slides-container", "slide-caption", "Sunset", "nav-button prev"} {
		if !strings.Contains(out, want) {
			t.Errorf("render carousel missing %q in:\n%s", want, out)
		}
	}

	// custom 合法
	out, _ = c.Render("image-layout", "---\nlayout: custom\ngrid: |\n  A A B\n  A A C\n---\n![[hero.jpg]]\n![[d1.jpg]]\n![[d2.jpg]]\n", "test.md")
	for _, want := range []string{"image-layouts-custom", "image-0 image-0 image-1", "image-0 image-0 image-2", "grid-area: image-0", "/images/hero.jpg"} {
		if !strings.Contains(out, want) {
			t.Errorf("render custom missing %q in:\n%s", want, out)
		}
	}

	// custom 非法 → 错误提示而非静默渲染
	out, _ = c.Render("image-layout", "---\nlayout: custom\ngrid: |\n  A B\n  A A\n---\n![[a.jpg]]\n![[b.jpg]]\n", "test.md")
	if !strings.Contains(out, "image-layouts-error") {
		t.Errorf("invalid custom grid should render error, got:\n%s", out)
	}

	// 空块 → 注释占位
	out, ok = c.Render("image-layout", "", "test.md")
	if !ok || !strings.Contains(out, "empty block") {
		t.Errorf("empty block: ok=%v out=%q", ok, out)
	}

	// 未知布局
	out, _ = c.Render("image-layout", "---\nlayout: zzz\n---\n![[a.jpg]]\n", "test.md")
	if !strings.Contains(out, "unknown layout") {
		t.Errorf("unknown layout should error, got:\n%s", out)
	}

	// legacy 对齐简写
	out, _ = c.Render("image-layout-right", "![[a.jpg]]\n", "test.md")
	for _, want := range []string{"image-layouts-align-right", "image-layouts-layout-single", "width:50%"} {
		if !strings.Contains(out, want) {
			t.Errorf("render right missing %q in:\n%s", want, out)
		}
	}

	// 非本插件 fence → 不处理
	if _, ok := c.Render("go", "fmt.Println(1)\n", "test.md"); ok {
		t.Error("non image-layout fence handled")
	}
}

func TestPadToSlots(t *testing.T) {
	imgs := []image{{link: "a"}, {link: "b"}, {link: "c"}, {link: "d"}}
	if got := padToSlots(imgs, 2); len(got) != 2 {
		t.Errorf("truncate: got %d", len(got))
	}
	if got := padToSlots(imgs, 6); len(got) != 6 || got[4].link != placeholderStyle {
		t.Errorf("pad: got %d items, placeholder wrong", len(got))
	}
}

func TestAssets(t *testing.T) {
	css := c.CSS()
	js := c.JS()
	if len(css) != 1 || len(js) != 2 {
		t.Fatalf("assets: css=%d js=%d (want css=1 js=2)", len(css), len(js))
	}
	if !strings.Contains(string(css[0]), ".image-layouts-layout-a") {
		t.Error("css missing grid styles")
	}
	if !strings.Contains(string(js[0]), "image-layout-carousel") {
		t.Error("js[0] missing carousel logic")
	}
	if !strings.Contains(string(js[1]), "IntersectionObserver") {
		t.Error("js[1] missing lazy loader logic")
	}
	if !strings.Contains(string(css[0]), "content-visibility") {
		t.Error("css missing content-visibility decoupling")
	}
}

func TestImageDimensionsFallback(t *testing.T) {
	if _, _, ok := imageDimensions("https://example.com/x.jpg"); ok {
		t.Error("remote url should not resolve")
	}
	if _, _, ok := imageDimensions("data:image/svg+xml,%3Csvg"); ok {
		t.Error("data uri should not resolve")
	}
	if _, _, ok := imageDimensions("/images/does-not-exist.png"); ok {
		t.Error("missing file should not resolve")
	}
}

func TestImgAttrs(t *testing.T) {
	first := imgAttrs(image{link: "a.jpg", width: 800, height: 600}, 0)
	for _, want := range []string{`loading="eager"`, `fetchpriority="high"`, `decoding="async"`, `width="800"`, `height="600"`} {
		if !strings.Contains(first, want) {
			t.Errorf("first img attrs missing %q: %s", want, first)
		}
	}
	other := imgAttrs(image{link: "b.jpg"}, 1)
	if !strings.Contains(other, `loading="lazy"`) || strings.Contains(other, "fetchpriority") {
		t.Errorf("lazy img attrs wrong: %s", other)
	}
	remote := imgAttrs(image{link: "https://x/y.jpg", external: true}, 2)
	if strings.Contains(remote, "width=") {
		t.Errorf("remote img should not have dimensions: %s", remote)
	}
	// 远程图即使作为首张也必须懒加载（不 eager），加载时机由 loader.js 控制
	remoteFirst := imgAttrs(image{link: "https://x/y.jpg", external: true}, 0)
	if strings.Contains(remoteFirst, "fetchpriority") || strings.Contains(remoteFirst, `loading="eager"`) {
		t.Errorf("remote first img must be lazy, got: %s", remoteFirst)
	}
	if !strings.Contains(remoteFirst, `loading="lazy"`) {
		t.Errorf("remote first img must have loading=lazy, got: %s", remoteFirst)
	}
}

func TestSrcAttr(t *testing.T) {
	if got := srcAttr(image{link: "a.jpg"}); got != "src" {
		t.Errorf("srcAttr(local) = %q, want src", got)
	}
	if got := srcAttr(image{link: "https://x/y.jpg", external: true}); got != "data-src" {
		t.Errorf("srcAttr(external) = %q, want data-src", got)
	}
}

func TestRenderExternalDataSrc(t *testing.T) {
	// 远程图渲染为 data-src（不产生 src），由 loader.js 按需加载
	out, ok := c.Render("image-layout-a",
		"![](https://example.com/x.jpg)\n![](https://example.com/y.jpg)\n", "test.md")
	if !ok {
		t.Fatal("external render not handled")
	}
	for _, want := range []string{`data-src="https://example.com/x.jpg"`, `data-src="https://example.com/y.jpg"`} {
		if !strings.Contains(out, want) {
			t.Errorf("external img missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, ` src="https://example.com`) {
		t.Errorf("external img should not emit src, got:\n%s", out)
	}
	// 本地图仍用 src
	out2, _ := c.Render("image-layout-a", "![[a.jpg]]\n![[b.jpg]]\n", "test.md")
	if !strings.Contains(out2, `src="/images/a.jpg"`) || strings.Contains(out2, `data-src="/images/`) {
		t.Errorf("local img should keep src, got:\n%s", out2)
	}
}
