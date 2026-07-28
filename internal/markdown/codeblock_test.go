package markdown

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

func TestMermaidBlocksAreEscaped(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(CodeBlockExt))
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
