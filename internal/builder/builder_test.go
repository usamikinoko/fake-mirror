package builder

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareOutputDirPreservesGitDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "public")
	gitDir := filepath.Join(root, ".git")

	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("create git dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "config"), []byte("gitdir"), 0644); err != nil {
		t.Fatalf("write git config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("stale"), 0644); err != nil {
		t.Fatalf("write stale artifact: %v", err)
	}

	if err := prepareOutputDir(root); err != nil {
		t.Fatalf("prepare output dir: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, ".git", "config")); err != nil {
		t.Fatalf("expected preserved git metadata: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "index.html")); !os.IsNotExist(err) {
		t.Fatalf("expected stale build artifact to be removed, got err=%v", err)
	}
}

func TestContainsMermaidFence(t *testing.T) {
	t.Run("detects mermaid code fences", func(t *testing.T) {
		body := "```mermaid\ngraph TD;\nA-->B\n```"
		if !containsMermaidFence(body) {
			t.Fatal("expected mermaid fence to be detected")
		}
	})

	t.Run("ignores plain text mentions", func(t *testing.T) {
		body := "This post talks about mermaid diagrams but does not embed one."
		if containsMermaidFence(body) {
			t.Fatal("did not expect plain text to trigger mermaid loading")
		}
	})
}

func TestRenderMarkdownPageMissingFileFallsBackToEmptyPage(t *testing.T) {
	fm, rendered, err := renderMarkdownPage(filepath.Join(t.TempDir(), "missing.md"), "About")
	if err != nil {
		t.Fatalf("renderMarkdownPage returned error: %v", err)
	}
	if fm.Title != "About" {
		t.Fatalf("expected fallback title, got %q", fm.Title)
	}
	if rendered.html != "" {
		t.Fatalf("expected empty rendered content, got %q", rendered.html)
	}
	if rendered.hasMermaid {
		t.Fatal("did not expect mermaid flag for missing file")
	}
}
