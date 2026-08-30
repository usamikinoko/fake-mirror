package markdown

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

func TestMermaidBlocksAreEscaped(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(Ext))
	source := "```mermaid\ngraph TD;\nA[<script>alert(1)</script>]-->B\n```"

	var buf bytes.Buffer
	if err := md.Convert([]byte(source), &buf); err != nil {
		t.Fatalf("render markdown: %v", err)
	}

	html := buf.String()
	if strings.Contains(html, "<script>alert(1)</script>") {
		t.Fatalf("expected mermaid payload to be escaped, got %q", html)
	}
	if !strings.Contains(html, "&lt;script&gt;alert(1)&lt;/script&gt;") {
		t.Fatalf("expected escaped mermaid payload, got %q", html)
	}
}

func TestImageRendererAddsLazyLoading(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(Ext))
	source := "![local](/images/a.png)\n\n![remote](https://example.com/x.png)\n"

	var buf bytes.Buffer
	if err := md.Convert([]byte(source), &buf); err != nil {
		t.Fatalf("render markdown: %v", err)
	}

	html := buf.String()
	for _, want := range []string{`src="/images/a.png"`, `src="https://example.com/x.png"`, `loading="lazy"`, `decoding="async"`} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered image missing %q:\n%s", want, html)
		}
	}
}
