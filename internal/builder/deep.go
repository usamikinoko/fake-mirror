package builder

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"rainhush/internal/config"

	"golang.org/x/crypto/pbkdf2"
)

const (
	deepSalt       = "rainhush.deep.salt"
	deepKey        = "rh.deep.2026"
	deepIterations = 100000
	deepTTLSeconds = 14400
	deepHeader     = "X-Deep-Auth"
)

type deepItem struct {
	Name  string `json:"name"`
	Title string `json:"title"`
	Date  string `json:"date"`
	URL   string `json:"url"`
}

func deepSlug(name string) string {
	return strings.Map(func(r rune) rune {
		if r >= 0x2010 && r <= 0x2015 || r == 0x2212 || r == 0xFF0D {
			return '-'
		}
		return r
	}, name)
}

func xorEncode(s, key string) string {
	var b strings.Builder
	b.Grow(len(s) * 2)
	for i := 0; i < len(s); i++ {
		fmt.Fprintf(&b, "%02x", s[i]^key[i%len(key)])
	}
	return b.String()
}

func deepPasswordHash() string {
	dk := pbkdf2.Key([]byte(config.Cfg.Deep.Password), []byte(deepSalt), deepIterations, 32, sha256.New)
	return hex.EncodeToString(dk)
}

func writeDeepJS(ctx *buildContext) error {
	src, err := os.ReadFile("static/js/deep.js")
	if err != nil {
		return err
	}
	cfg, err := json.Marshal(map[string]interface{}{
		"t": deepTTLSeconds,
		"h": deepHeader,
		"i": deepIterations,
		"s": xorEncode(deepSalt, deepKey),
		"p": xorEncode(deepPasswordHash(), deepKey),
		"k": xorEncode(config.Cfg.Deep.Sign, deepKey),
	})
	if err != nil {
		return err
	}
	content := strings.ReplaceAll(string(src), "__DEEP_CFG__", string(cfg))
	hash := contentHash([]byte(content))
	ctx.deepJS = "/static/deep." + hash + ".js"
	return os.WriteFile(filepath.Join("public", ctx.deepJS[1:]), []byte(content), 0644)
}

func writeDeepFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func deepHomeFragment(home renderedMarkdown) string {
	return `<div class="post-body">` + home.html + `</div>`
}

func deepPostFragment(p *Post) string {
	var b strings.Builder
	b.WriteString(`<h2 class="deep-post-title">` + template.HTMLEscapeString(p.Title) + `</h2>`)
	b.WriteString(`<div class="post-meta-top"><div class="post-meta-row">`)
	b.WriteString(`<time class="post-date">Published on ` + template.HTMLEscapeString(p.Date) + `</time>`)
	if p.UpdatedAt != "" {
		b.WriteString(`<time class="post-updated">Updated at ` + template.HTMLEscapeString(p.UpdatedAt) + `</time>`)
	}
	b.WriteString(`</div></div><div class="post-body">` + string(p.Content) + `</div>`)
	return b.String()
}

func (ctx *buildContext) renderDeep() error {
	if config.Cfg == nil || !config.Cfg.Deep.Enabled {
		return nil
	}
	if strings.TrimSpace(config.Cfg.Deep.Password) == "" {
		return fmt.Errorf("deep.password is required when deep.enabled is true")
	}
	if strings.TrimSpace(config.Cfg.Deep.Sign) == "" {
		return fmt.Errorf("deep.sign is required when deep.enabled is true")
	}

	if err := writeDeepJS(ctx); err != nil {
		return err
	}

	homeFM, home, err := renderMarkdownPage("content/deep/home/home.md", "Deep")
	if err != nil {
		return err
	}
	if err := writeDeepFile(filepath.Join("public", "deep", "data", "home.html"), []byte(deepHomeFragment(home))); err != nil {
		return err
	}

	posts, err := loadPosts("content/deep/posts")
	if err != nil {
		return err
	}
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].PublishedAt.After(posts[j].PublishedAt)
	})

	items := make([]deepItem, 0, len(posts))
	for _, p := range posts {
		slug := deepSlug(p.Filename)
		if err := writeDeepFile(filepath.Join("public", "deep", "data", "posts", slug+".html"), []byte(deepPostFragment(p))); err != nil {
			return err
		}
		items = append(items, deepItem{
			Name:  slug,
			Title: p.Title,
			Date:  p.Date,
			URL:   "/deep/articles/" + slug + "/",
		})
	}

	index, err := json.Marshal(map[string]interface{}{"items": items})
	if err != nil {
		return err
	}
	if err := writeDeepFile(filepath.Join("public", "deep", "data", "index.json"), index); err != nil {
		return err
	}

	if err := ctx.renderDeepIndex(homeFM); err != nil {
		return err
	}
	for _, p := range posts {
		if err := ctx.renderDeepPost(p); err != nil {
			return err
		}
	}
	return ctx.renderAuthPage()
}

func (ctx *buildContext) renderDeepIndex(home *Frontmatter) error {
	tmpl, err := ctx.cloneTmpl()
	if err != nil {
		return err
	}
	if _, err := tmpl.ParseFiles("templates/pages/deep.html"); err != nil {
		return err
	}
	title := strings.TrimSpace(home.Title)
	if title == "" {
		title = "Deep"
	}
	canonical := strings.TrimRight(config.Cfg.Site.URL, "/") + "/deep/"
	return ctx.writeHTML(tmpl, filepath.Join("public", "deep", "index.html"), ctx.pageData(map[string]interface{}{
		"Title":        title,
		"CanonicalURL": canonical,
		"NoIndex":      true,
		"Nav":          navState(""),
		"DeepJS":       ctx.deepJS,
	}))
}

func (ctx *buildContext) renderDeepPost(p *Post) error {
	tmpl, err := ctx.cloneTmpl()
	if err != nil {
		return err
	}
	if _, err := tmpl.ParseFiles("templates/pages/deep-post.html"); err != nil {
		return err
	}
	slug := deepSlug(p.Filename)
	canonical := strings.TrimRight(config.Cfg.Site.URL, "/") + "/deep/articles/" + slug + "/"
	return ctx.writeHTML(tmpl, filepath.Join("public", "deep", "articles", slug, "index.html"), ctx.pageData(map[string]interface{}{
		"Title":        p.Title,
		"CanonicalURL": canonical,
		"NoIndex":      true,
		"Nav":          navState(""),
		"DeepJS":       ctx.deepJS,
	}))
}

func (ctx *buildContext) renderAuthPage() error {
	tmpl, err := ctx.cloneTmpl()
	if err != nil {
		return err
	}
	if _, err := tmpl.ParseFiles("templates/pages/auth.html"); err != nil {
		return err
	}
	return ctx.writeHTML(tmpl, filepath.Join("public", "auth.html"), ctx.pageData(map[string]interface{}{
		"Title":   "Private Access",
		"NoIndex": true,
		"Nav":     navState(""),
		"DeepJS":  ctx.deepJS,
	}))
}
