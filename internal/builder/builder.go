package builder

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"html/template"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"rainhush/internal/config"
	"rainhush/internal/markdown"
	"rainhush/pkg/extension"

	"github.com/yuin/goldmark"
	goldmarkext "github.com/yuin/goldmark/extension"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"
	"gopkg.in/yaml.v3"
)

const articlesPerPage = 10

type Frontmatter struct {
	Title     string `yaml:"title"`
	Author    string `yaml:"author"`
	Date      string `yaml:"date"`
	UpdatedAt string `yaml:"updated_at"`
	Location  string `yaml:"location"`
	Avatar    string `yaml:"avatar"`
	Cover     string `yaml:"cover"`
	Category  string `yaml:"category"`
}

type Post struct {
	Frontmatter
	Content     template.HTML
	Filename    string
	Excerpt     string
	PublishedAt time.Time
}

type navMap map[string]string

type pageItem struct {
	Number  int
	URL     string
	Current bool
}

type heatmapCell struct {
	Level int
	Date  string
	Count int
}

type heatmapMonth struct {
	Label string
	Col   int
}

type renderedMarkdown struct {
	html string
}

type buildContext struct {
	commonTmpl *template.Template
	bundleCSS  string
	bundleJS   string
}

// Markdown 内容由站点作者维护，允许其使用表格、徽章等原始 HTML。
// 不应用于不受信任的用户输入。
var md = goldmark.New(
	goldmark.WithExtensions(
		goldmarkext.Table,
		markdown.Ext,
	),
	goldmark.WithRendererOptions(goldmarkhtml.WithUnsafe()),
)

func newBuildContext() (*buildContext, error) {
	tmpl, err := template.ParseFiles(
		"templates/layouts/layout.html",
		"templates/components/header.html",
		"templates/components/footer.html",
		"templates/components/heatmap.html",
	)
	if err != nil {
		return nil, fmt.Errorf("parse shared templates: %w", err)
	}

	return &buildContext{commonTmpl: tmpl}, nil
}

func (ctx *buildContext) cloneTmpl() (*template.Template, error) {
	return ctx.commonTmpl.Clone()
}

func navState(active string) navMap {
	items := []string{"Home", "Articles", "Friends", "About"}
	state := make(navMap, len(items))
	for _, item := range items {
		if item == active {
			state[item] = "active"
			continue
		}
		state[item] = ""
	}
	return state
}

var (
	navHome     = navState("Home")
	navArticles = navState("Articles")
	navFriends  = navState("Friends")
	navAbout    = navState("About")
)

func Build() error {
	for _, name := range extension.Default.Names() {
		extension.SetEnabled(name, config.ExtensionEnabled(name))
	}
	if names := extension.Default.String(); names != "" {
		fmt.Printf("Extensions: %s\n", names)
	}
	if err := prepareOutputDir("public"); err != nil {
		return fmt.Errorf("prepare public directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join("public", "static"), 0755); err != nil {
		return fmt.Errorf("create public/static: %w", err)
	}

	if err := copyDir("static", "public/static"); err != nil {
		return err
	}

	ctx, err := newBuildContext()
	if err != nil {
		return err
	}
	if err := ctx.bundleAssets(); err != nil {
		return fmt.Errorf("bundle assets: %w", err)
	}

	posts, err := loadPosts(filepath.Join("content", "posts"))
	if err != nil {
		return err
	}
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].PublishedAt.After(posts[j].PublishedAt)
	})

	if err := ctx.renderAll(posts); err != nil {
		return err
	}
	if err := ctx.renderAbout(); err != nil {
		return err
	}
	if err := ctx.renderFriends(); err != nil {
		return err
	}
	if err := ctx.renderDeep(); err != nil {
		return err
	}
	if err := renderSitemap(posts); err != nil {
		return err
	}
	if err := renderRSS(posts); err != nil {
		return err
	}

	for _, f := range []string{"vercel.json", "robots.txt"} {
		if src, err := os.ReadFile(f); err == nil {
			if err := os.WriteFile(filepath.Join("public", f), src, 0644); err != nil {
				return fmt.Errorf("write %s: %w", f, err)
			}
		}
	}

	fmt.Println("Site built to public/")
	return nil
}

func prepareOutputDir(root string) error {
	gitDir := filepath.Join(root, ".git")
	stashDir := filepath.Join(filepath.Dir(root), "."+filepath.Base(root)+".git.keep")

	if err := os.RemoveAll(stashDir); err != nil {
		return fmt.Errorf("cleanup git stash: %w", err)
	}

	hasGitDir := false
	if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
		hasGitDir = true
		if err := os.Rename(gitDir, stashDir); err != nil {
			return fmt.Errorf("preserve %s: %w", gitDir, err)
		}
	}

	restoreGitDir := func() error {
		if !hasGitDir {
			return nil
		}
		if err := os.MkdirAll(root, 0755); err != nil {
			return fmt.Errorf("recreate %s: %w", root, err)
		}
		if err := os.Rename(stashDir, gitDir); err != nil {
			return fmt.Errorf("restore %s: %w", gitDir, err)
		}
		hasGitDir = false
		return nil
	}

	if err := os.RemoveAll(root); err != nil {
		_ = restoreGitDir()
		return fmt.Errorf("remove %s: %w", root, err)
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		_ = restoreGitDir()
		return fmt.Errorf("create %s: %w", root, err)
	}
	if err := restoreGitDir(); err != nil {
		return err
	}

	return nil
}

func loadPosts(postsDir string) ([]*Post, error) {
	var posts []*Post

	entries, err := os.ReadDir(postsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return posts, nil
		}
		return nil, fmt.Errorf("read posts directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		post, err := parsePost(filepath.Join(postsDir, entry.Name()))
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}

	return posts, nil
}

func (ctx *buildContext) renderAll(posts []*Post) error {
	tmpl, err := ctx.cloneTmpl()
	if err != nil {
		return err
	}
	if _, err := tmpl.ParseFiles("templates/pages/post.html"); err != nil {
		return err
	}
	for _, p := range posts {
		if err := ctx.renderPost(tmpl, p); err != nil {
			return err
		}
	}

	tmpl, err = ctx.cloneTmpl()
	if err != nil {
		return err
	}
	if _, err := tmpl.ParseFiles("templates/pages/index.html"); err != nil {
		return err
	}
	if err := ctx.renderIndex(tmpl, posts); err != nil {
		return err
	}

	tmpl, err = ctx.cloneTmpl()
	if err != nil {
		return err
	}
	if _, err := tmpl.ParseFiles("templates/pages/articles.html"); err != nil {
		return err
	}
	if err := ctx.renderArticles(tmpl, posts); err != nil {
		return err
	}

	return nil
}

func parsePost(path string) (*Post, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	markdown.SetDocument(path)
	defer markdown.SetDocument("")

	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	fm, body, err := parseFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	publishedAt, err := validatePostFrontmatter(path, fm)
	if err != nil {
		return nil, err
	}

	rendered, err := renderMarkdown(body)
	if err != nil {
		return nil, err
	}

	filename := strings.TrimSuffix(filepath.Base(path), ".md")
	return &Post{
		Frontmatter: *fm,
		Content:     template.HTML(rendered.html),
		Filename:    filename,
		Excerpt:     extractExcerpt(body),
		PublishedAt: publishedAt,
	}, nil
}

func validatePostFrontmatter(path string, fm *Frontmatter) (time.Time, error) {
	if strings.TrimSpace(fm.Title) == "" {
		return time.Time{}, fmt.Errorf("parse %s: missing required frontmatter field %q", path, "title")
	}
	if strings.TrimSpace(fm.Date) == "" {
		return time.Time{}, fmt.Errorf("parse %s: missing required frontmatter field %q", path, "date")
	}

	publishedAt, err := parseISODate(fm.Date)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse %s: invalid date %q: %w", path, fm.Date, err)
	}
	if fm.UpdatedAt != "" {
		if _, err := parseISODate(fm.UpdatedAt); err != nil {
			return time.Time{}, fmt.Errorf("parse %s: invalid updated_at %q: %w", path, fm.UpdatedAt, err)
		}
	}

	return publishedAt, nil
}

func parseISODate(raw string) (time.Time, error) {
	return time.Parse("2006-01-02", raw)
}

func splitFrontmatter(content string) (string, string) {
	content = strings.TrimPrefix(content, "\uFEFF")
	if !strings.HasPrefix(content, "---\n") {
		return "", content
	}

	rest := content[4:]
	if idx := strings.Index(rest, "\n---\n"); idx >= 0 {
		return rest[:idx], rest[idx+5:]
	}
	// 结尾 `---` 后无换行的 EOF 情况也视为合法 front matter 闭合
	if strings.HasSuffix(rest, "\n---") {
		return rest[:len(rest)-4], ""
	}
	return "", content
}

func parseFrontmatter(content string) (*Frontmatter, string, error) {
	fm := &Frontmatter{}
	raw, body := splitFrontmatter(content)

	if raw != "" {
		if err := yaml.Unmarshal([]byte(raw), fm); err != nil {
			return nil, "", err
		}
	}

	return fm, body, nil
}

func renderMarkdown(body string) (renderedMarkdown, error) {
	var buf bytes.Buffer
	if err := md.Convert([]byte(body), &buf); err != nil {
		return renderedMarkdown{}, err
	}

	return renderedMarkdown{html: buf.String()}, nil
}

func (ctx *buildContext) renderPost(tmpl *template.Template, post *Post) error {
	dir := filepath.Join("public", "articles", post.Filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	canonicalURL := strings.TrimRight(config.Cfg.Site.URL, "/") + "/articles/" + post.Filename + "/"
	return ctx.writeHTML(tmpl, filepath.Join(dir, "index.html"), ctx.pageData(map[string]interface{}{
		"CanonicalURL": canonicalURL,
		"Title":        post.Title,
		"Author":       post.Author,
		"Date":         post.Date,
		"UpdatedAt":    post.UpdatedAt,
		"Location":     post.Location,
		"Avatar":       post.Avatar,
		"Cover":        post.Cover,
		"Content":      post.Content,
		"Nav":          navArticles,
	}))
}

func (ctx *buildContext) renderIndex(tmpl *template.Template, posts []*Post) error {
	perCategory := 3

	var techPosts, lifePosts, liteTechPosts, othersPosts []*Post
	for _, p := range posts {
		switch p.Category {
		case "technology":
			if len(techPosts) < perCategory {
				techPosts = append(techPosts, p)
			}
		case "life":
			if len(lifePosts) < perCategory {
				lifePosts = append(lifePosts, p)
			}
		case "lite-tech":
			if len(liteTechPosts) < perCategory {
				liteTechPosts = append(liteTechPosts, p)
			}
		default:
			if len(othersPosts) < perCategory {
				othersPosts = append(othersPosts, p)
			}
		}
	}

	cells, dl, ml, ht := computeHeatmap(posts)

	return ctx.writeHTML(tmpl, filepath.Join("public", "index.html"), ctx.pageData(map[string]interface{}{
		"Title": "Home",
		"Home": map[string]string{
			"Title":    config.Cfg.Home.Title,
			"SubTitle": config.Cfg.Home.SubTitle,
			"Avatar":   config.Cfg.Home.Avatar,
			"Owner":    config.Cfg.Home.Owner,
		},
		"TechPosts":        techPosts,
		"LifePosts":        lifePosts,
		"LiteTechPosts":    liteTechPosts,
		"OthersPosts":      othersPosts,
		"Nav":              navHome,
		"HeatmapCells":     cells,
		"HeatmapDayLabels": dl,
		"HeatmapMonths":    ml,
		"HeatmapTotal":     ht,
		"CanonicalURL":     strings.TrimRight(config.Cfg.Site.URL, "/") + "/",
	}))
}

func (ctx *buildContext) renderArticles(tmpl *template.Template, posts []*Post) error {
	totalPages := max(int(math.Ceil(float64(len(posts))/float64(articlesPerPage))), 1)

	for page := 1; page <= totalPages; page++ {
		start := (page - 1) * articlesPerPage
		end := min(start+articlesPerPage, len(posts))

		var outPath string
		var canonicalURL string
		if page == 1 {
			outPath = filepath.Join("public", "articles.html")
			canonicalURL = strings.TrimRight(config.Cfg.Site.URL, "/") + "/articles.html"
		} else {
			dir := filepath.Join("public", "articles", "page", strconv.Itoa(page))
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
			outPath = filepath.Join(dir, "index.html")
			canonicalURL = strings.TrimRight(config.Cfg.Site.URL, "/") + "/articles/page/" + strconv.Itoa(page) + "/"
		}

		pageItems := make([]pageItem, 0, totalPages)
		for i := 1; i <= totalPages; i++ {
			url := "/articles.html"
			if i > 1 {
				url = "/articles/page/" + strconv.Itoa(i) + "/"
			}
			pageItems = append(pageItems, pageItem{
				Number:  i,
				URL:     url,
				Current: i == page,
			})
		}

		var prevURL, nextURL string
		if page > 1 {
			if page-1 == 1 {
				prevURL = "/articles.html"
			} else {
				prevURL = "/articles/page/" + strconv.Itoa(page-1) + "/"
			}
		}
		if page < totalPages {
			nextURL = "/articles/page/" + strconv.Itoa(page+1) + "/"
		}

		if err := ctx.writeHTML(tmpl, outPath, ctx.pageData(map[string]interface{}{
			"Title":        "Articles",
			"Posts":        posts[start:end],
			"Page":         page,
			"TotalPages":   totalPages,
			"TotalPosts":   len(posts),
			"HasPrev":      page > 1,
			"PrevURL":      prevURL,
			"HasNext":      page < totalPages,
			"NextURL":      nextURL,
			"PageItems":    pageItems,
			"CanonicalURL": canonicalURL,
			"Nav":          navArticles,
		})); err != nil {
			return err
		}
	}

	return nil
}

func (ctx *buildContext) renderAbout() error {
	tmpl, err := ctx.cloneTmpl()
	if err != nil {
		return err
	}
	if _, err := tmpl.ParseFiles("templates/pages/about.html"); err != nil {
		return err
	}

	for _, p := range []struct{ out, src, lang string }{
		{"about.html", "content/about/about_CN.md", "zh-CN"},
		{"about_EN.html", "content/about/about_EN.md", "en"},
	} {
		if err := ctx.renderAboutPage(tmpl, p.out, p.src, p.lang); err != nil {
			return err
		}
	}
	return nil
}

func (ctx *buildContext) renderAboutPage(tmpl *template.Template, out, src, lang string) error {
	fm, rendered, err := renderMarkdownPage(src, "About")
	if err != nil {
		return err
	}
	canonicalURL := strings.TrimRight(config.Cfg.Site.URL, "/") + "/" + out
	return ctx.writeHTML(tmpl, filepath.Join("public", out), ctx.pageData(map[string]interface{}{
		"CanonicalURL": canonicalURL,
		"Title":        fm.Title,
		"Content":      template.HTML(rendered.html),
		"Nav":          navAbout,
		"Lang":         lang,
	}))
}

func (ctx *buildContext) renderFriends() error {
	fm, rendered, err := renderMarkdownPage("content/friends/friends.md", "Friends")
	if err != nil {
		return err
	}

	tmpl, err := ctx.cloneTmpl()
	if err != nil {
		return err
	}
	if _, err := tmpl.ParseFiles("templates/pages/friends.html"); err != nil {
		return err
	}

	canonicalURL := strings.TrimRight(config.Cfg.Site.URL, "/") + "/friends.html"
	return ctx.writeHTML(tmpl, filepath.Join("public", "friends.html"), ctx.pageData(map[string]interface{}{
		"CanonicalURL": canonicalURL,
		"Title":        fm.Title,
		"Content":      template.HTML(rendered.html),
		"Nav":          navFriends,
	}))
}

func renderMarkdownPage(path, fallbackTitle string) (*Frontmatter, renderedMarkdown, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Frontmatter{Title: fallbackTitle}, renderedMarkdown{}, nil
		}
		return nil, renderedMarkdown{}, err
	}

	markdown.SetDocument(path)
	defer markdown.SetDocument("")

	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	fm, body, err := parseFrontmatter(content)
	if err != nil {
		return nil, renderedMarkdown{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if fm.Title == "" {
		fm.Title = fallbackTitle
	}

	rendered, err := renderMarkdown(body)
	if err != nil {
		return nil, renderedMarkdown{}, err
	}

	return fm, rendered, nil
}

func computeHeatmap(posts []*Post) (cells []heatmapCell, dayLabels []string, months []heatmapMonth, total int) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var start time.Time
	startSunday := time.Sunday
	if today.Weekday() >= startSunday {
		start = today.AddDate(0, 0, -int(today.Weekday()-startSunday)-364)
	} else {
		start = today.AddDate(0, 0, -int(today.Weekday()+7-startSunday)-364)
	}

	counts := map[string]int{}
	for _, p := range posts {
		date := p.PublishedAt.Format("2006-01-02")
		counts[date]++
		total++
	}

	// Column-major: 53 weeks * 7 days = 371 cells
	cells = make([]heatmapCell, 53*7)
	for w := 0; w < 53; w++ {
		for d := 0; d < 7; d++ {
			day := start.AddDate(0, 0, w*7+d)
			date := day.Format("2006-01-02")
			c := counts[date]
			cell := heatmapCell{
				Date:  day.Format("Jan 2, 2006"),
				Count: c,
			}
			switch {
			case c >= 4:
				cell.Level = 4
			case c == 3:
				cell.Level = 3
			case c == 2:
				cell.Level = 2
			case c == 1:
				cell.Level = 1
			}
			cells[w*7+d] = cell
		}
	}

	dayLabels = []string{"", "Mon", "", "Wed", "", "Fri", ""}

	for w := 0; w < 53; w++ {
		d := start.AddDate(0, 0, w*7)
		m := d.Format("Jan")
		if len(months) == 0 || months[len(months)-1].Label != m {
			months = append(months, heatmapMonth{
				Label: m,
				Col:   w,
			})
		}
	}
	return
}

func renderSitemap(posts []*Post) error {
	if config.Cfg.Site.URL == "" {
		return nil
	}
	base := strings.TrimRight(config.Cfg.Site.URL, "/")

	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	buf.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)

	add := func(loc, lastmod, priority string) {
		buf.WriteString("<url><loc>" + escapeXML(base+loc) + "</loc>")
		if lastmod != "" {
			buf.WriteString("<lastmod>" + escapeXML(lastmod) + "</lastmod>")
		}
		if priority != "" {
			buf.WriteString("<priority>" + escapeXML(priority) + "</priority>")
		}
		buf.WriteString("</url>")
	}

	add("/", "", "1.0")
	add("/articles.html", "", "0.9")
	add("/friends.html", "", "0.6")
	add("/about.html", "", "0.6")
	add("/about_EN.html", "", "0.6")

	totalPages := max(int(math.Ceil(float64(len(posts))/float64(articlesPerPage))), 1)
	for page := 2; page <= totalPages; page++ {
		add("/articles/page/"+strconv.Itoa(page)+"/", "", "0.6")
	}
	for _, p := range posts {
		add("/articles/"+p.Filename+"/", p.Date, "0.7")
	}

	buf.WriteString("</urlset>")
	return os.WriteFile(filepath.Join("public", "sitemap.xml"), buf.Bytes(), 0644)
}

func renderRSS(posts []*Post) error {
	if config.Cfg.Site.URL == "" {
		return nil
	}
	base := strings.TrimRight(config.Cfg.Site.URL, "/")

	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	buf.WriteString(`<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom"><channel>`)
	buf.WriteString("<title>" + escapeXML(siteTitle()) + "</title>")
	buf.WriteString("<link>" + escapeXML(base) + "</link>")
	buf.WriteString("<description>" + escapeXML(config.Cfg.Site.Description) + "</description>")
	buf.WriteString("<lastBuildDate>" + escapeXML(time.Now().Format(time.RFC1123Z)) + "</lastBuildDate>")
	buf.WriteString("<atom:link href=\"" + escapeXML(base+"/feed.xml") + "\" rel=\"self\" type=\"application/rss+xml\"/>")

	for _, p := range posts {
		buf.WriteString("<item>")
		buf.WriteString("<title>" + escapeXML(p.Title) + "</title>")
		buf.WriteString("<link>" + escapeXML(base+"/articles/"+p.Filename+"/") + "</link>")
		buf.WriteString("<guid isPermaLink=\"true\">" + escapeXML(base+"/articles/"+p.Filename+"/") + "</guid>")
		buf.WriteString("<pubDate>" + escapeXML(p.PublishedAt.Format(time.RFC1123Z)) + "</pubDate>")
		if p.Excerpt != "" {
			buf.WriteString("<description><![CDATA[" + safeCDATA(p.Excerpt) + "]]></description>")
		}
		buf.WriteString("</item>")
	}

	buf.WriteString("</channel></rss>")
	return os.WriteFile(filepath.Join("public", "feed.xml"), buf.Bytes(), 0644)
}

func siteTitle() string {
	if config.Cfg != nil && strings.TrimSpace(config.Cfg.Home.Title) != "" {
		return strings.TrimSpace(config.Cfg.Home.Title)
	}
	return "Fake Mirror"
}

func safeCDATA(value string) string {
	return strings.ReplaceAll(value, "]]>", "]]]]><![CDATA[>")
}

func (ctx *buildContext) pageData(extra map[string]interface{}) map[string]interface{} {
	data := map[string]interface{}{
		"Site":      config.Cfg.Site,
		"SiteName":  siteTitle(),
		"BundleCSS": ctx.bundleCSS,
		"BundleJS":  ctx.bundleJS,
		"Year":      time.Now().Year(),
	}
	for k, v := range extra {
		data[k] = v
	}
	return data
}

func (ctx *buildContext) writeHTML(tmpl *template.Template, path string, data map[string]interface{}) error {
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "layout", data); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, minifyHTML(buf.Bytes()), 0644)
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			if relPath == "css" || relPath == "js" {
				return filepath.SkipDir
			}
			return os.MkdirAll(dstPath, info.Mode())
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dstPath, data, info.Mode())
	})
}

func escapeXML(value string) string {
	var buf bytes.Buffer
	if err := xml.EscapeText(&buf, []byte(value)); err != nil {
		return value
	}
	return buf.String()
}

var cssFiles = []string{
	"static/css/layout.css",
	"static/css/components/header.css",
	"static/css/components/chroma.css",
	"static/css/pages/index.css",
	"static/css/pages/post.css",
	"static/css/components/code.css",
	"static/css/pages/articles.css",
	"static/css/pages/friends.css",
	"static/css/components/heatmap.css",
	"static/css/pages/deep.css",
}

var jsFiles = []string{
	"static/js/lifecycle.js",
	"static/js/code.js",
	"static/js/mermaid.js",
	"static/js/header.js",
	"static/js/toc.js",
	"static/js/rain.js",
	"static/js/navigation.js",
}

func (ctx *buildContext) bundleAssets() error {
	var buf bytes.Buffer
	for _, p := range cssFiles {
		data, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("read %s: %w", p, err)
		}
		buf.Write(data)
		buf.WriteByte('\n')
	}
	for _, c := range extension.Default.CSS() {
		buf.Write(c)
		buf.WriteByte('\n')
	}

	minifiedCSS := minifyCSS(buf.Bytes())
	cssHash := contentHash(minifiedCSS)
	ctx.bundleCSS = "/static/bundle." + cssHash + ".css"
	if err := os.WriteFile(filepath.Join("public", ctx.bundleCSS[1:]), minifiedCSS, 0644); err != nil {
		return err
	}

	buf.Reset()
	for _, p := range jsFiles {
		data, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("read %s: %w", p, err)
		}
		buf.Write(data)
		buf.Write([]byte{';', '\n'})
	}
	for _, j := range extension.Default.JS() {
		buf.Write(j)
		buf.Write([]byte{';', '\n'})
	}

	jsContent := buf.Bytes()
	jsHash := contentHash(jsContent)
	ctx.bundleJS = "/static/bundle." + jsHash + ".js"
	if err := os.WriteFile(filepath.Join("public", ctx.bundleJS[1:]), jsContent, 0644); err != nil {
		return err
	}

	return nil
}

func contentHash(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])[:8]
}

func minifyCSS(data []byte) []byte {
	s := string(data)
	s = stripCSSComments(s)

	var b strings.Builder
	b.Grow(len(s))
	inWhitespace := false
	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\t' || r == ' ' {
			if !inWhitespace {
				b.WriteByte(' ')
				inWhitespace = true
			}
			continue
		}
		if r == '{' || r == '}' || r == ';' || r == ':' || r == ',' {
			inWhitespace = false
			b.WriteRune(r)
			continue
		}
		inWhitespace = false
		b.WriteRune(r)
	}
	return []byte(strings.TrimSpace(b.String()))
}

func stripCSSComments(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	i := 0
	for i < len(s) {
		if i+1 < len(s) && s[i] == '/' && s[i+1] == '*' {
			i += 2
			for i+1 < len(s) && !(s[i] == '*' && s[i+1] == '/') {
				i++
			}
			i += 2
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

var excerptMoreRe = regexp.MustCompile(`<!--\s*more\s*-->`)

func minifyHTML(data []byte) []byte {
	s := string(data)
	blocks := make([]string, 0)

	s = htmlBlockRe.ReplaceAllStringFunc(s, func(match string) string {
		blocks = append(blocks, match)
		return fmt.Sprintf(blockSentinelFmt, len(blocks)-1)
	})

	s = htmlTagWS.ReplaceAllString(s, "><")

	for i, block := range blocks {
		s = strings.Replace(s, fmt.Sprintf(blockSentinelFmt, i), block, 1)
	}

	return []byte(strings.TrimSpace(s))
}

func extractExcerpt(body string) string {
	if loc := excerptMoreRe.FindStringIndex(body); loc != nil {
		idx := loc[0]
		excerptBody := strings.TrimSpace(body[:idx])

		var buf bytes.Buffer
		if err := md.Convert([]byte(excerptBody), &buf); err != nil {
			return ""
		}

		text := stripTags(buf.String())
		return truncateText(strings.TrimSpace(text), 200)
	}

	var buf bytes.Buffer
	if err := md.Convert([]byte(body), &buf); err != nil {
		return ""
	}
	html := buf.String()
	if idx := strings.Index(html, "</p>"); idx != -1 && idx < 300 {
		if pStart := strings.LastIndex(html[:idx], "<p>"); pStart != -1 {
			text := stripTags(html[pStart : idx+5])
			return truncateText(strings.TrimSpace(text), 200)
		}
	}
	return ""
}

func stripTags(html string) string {
	var buf bytes.Buffer
	inTag := false
	for _, r := range html {
		if r == '<' {
			inTag = true
		} else if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			buf.WriteRune(r)
		}
	}
	return buf.String()
}

func truncateText(text string, maxLen int) string {
	runes := []rune(text)
	if len(runes) <= maxLen {
		return text
	}
	return string(runes[:maxLen]) + "...."
}

var (
	htmlBlockRe = regexp.MustCompile(`(?is)<pre\b[^>]*>.*?</pre>|<textarea\b[^>]*>.*?</textarea>|<script\b[^>]*>.*?</script>|<style\b[^>]*>.*?</style>`)
	htmlTagWS   = regexp.MustCompile(`>\s+<`)
)

var blockSentinelFmt = "\x01BLOCK%04d\x01"
