package markdown

import (
	"bytes"
	stdhtml "html"
	"strings"

	"rainhush/internal/markdown/components"
	"rainhush/internal/plugins"

	_ "rainhush/internal/markdown/components/imagelayout"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type CodeBlock struct{}

var CodeBlockExt = &CodeBlock{}

func (e *CodeBlock) Extend(m goldmark.Markdown) {
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(&codeBlockRenderer{}, 0),
	))
}

type codeBlockRenderer struct{}

func (r *codeBlockRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindFencedCodeBlock, r.renderFencedCodeBlock)
}

func (r *codeBlockRenderer) renderFencedCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.FencedCodeBlock)
	if !entering {
		return ast.WalkContinue, nil
	}

	lang := string(n.Language(source))

	if lang == "mermaid" {
		w.WriteString(`<pre class="mermaid">`)
		lines := n.Lines()
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			w.WriteString(stdhtml.EscapeString(string(line.Value(source))))
		}
		w.WriteString(`</pre>`)
		return ast.WalkSkipChildren, nil
	}

	var rawCode []byte
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		v := line.Value(source)
		if i == 0 {
			v = bytes.TrimLeft(v, "\n\r\t ")
		}
		rawCode = append(rawCode, v...)
	}

	rawCodeStr := strings.TrimLeft(string(rawCode), "\n\r\t ")

	doc := Document()
	if out, ok := components.RenderFence(lang, rawCodeStr, doc); ok {
		w.WriteString(out)
		return ast.WalkSkipChildren, nil
	}
	if out, ok := plugins.RenderFence(lang, rawCodeStr, plugins.Context{Document: doc}); ok {
		w.WriteString(out)
		return ast.WalkSkipChildren, nil
	}

	w.WriteString(`<div class="code-block">`)
	w.WriteString(`<div class="code-block-header">`)
	w.WriteString(`<div class="code-block-header-left">`)
	w.WriteString(`<span class="code-block-lang">`)
	if lang != "" {
		w.WriteString(stdhtml.EscapeString(lang))
	} else {
		w.WriteString(`code`)
	}
	w.WriteString(`</span>`)
	w.WriteString(`</div>`)
	w.WriteString(`<div class="code-block-header-right">`)
	w.WriteString(`<button class="copy-btn" onclick="copyCode(this)" title="Copy code">`)
	w.WriteString(`<svg class="copy-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>`)
	w.WriteString(`<span class="copy-text">Copy</span>`)
	w.WriteString(`</button>`)
	w.WriteString(`</div>`)
	w.WriteString(`</div>`)
	w.WriteString(`<pre><code`)

	if lang != "" {
		escapedLang := stdhtml.EscapeString(lang)
		w.WriteString(` class="language-` + escapedLang + `"`)

		lexer := lexers.Get(lang)
		if lexer == nil {
			lexer = lexers.Analyse(rawCodeStr)
		}
		if lexer == nil {
			lexer = lexers.Fallback
		}
		lexer = chroma.Coalesce(lexer)

		style := styles.Get("monokai")
		if style == nil {
			style = styles.Fallback
		}

		formatter := chromahtml.New(chromahtml.WithClasses(true), chromahtml.TabWidth(4), chromahtml.PreventSurroundingPre(true))
		iterator, err := lexer.Tokenise(nil, rawCodeStr)
		if err == nil {
			var buf bytes.Buffer
			if err := formatter.Format(&buf, style, iterator); err == nil {
				w.WriteString(`>`)
				highlighted := strings.TrimSpace(buf.String())
				w.WriteString(highlighted)
				w.WriteString(`</code></pre>`)
				w.WriteString(`</div>`)
				return ast.WalkSkipChildren, nil
			}
		}
	}

	w.WriteString(`>`)
	w.WriteString(stdhtml.EscapeString(strings.TrimRight(rawCodeStr, "\n")))
	w.WriteString(`</code></pre>`)
	w.WriteString(`</div>`)

	return ast.WalkSkipChildren, nil
}
