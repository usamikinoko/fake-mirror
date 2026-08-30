package betterfriends

import (
	"strings"
	"testing"

	"rainhush/pkg/extension"
)

func TestRenderFenceBuildsFriendCards(t *testing.T) {
	out, handled, err := (&plugin{}).RenderFence(fenceLanguage, "- Alice: [https://alice.example/](https://alice.example/)\n- Bob: [http://bob.example/](http://bob.example/)", extension.Context{})
	if err != nil {
		t.Fatalf("RenderFence returned error: %v", err)
	}
	if !handled {
		t.Fatal("expected better-friends fence to be handled")
	}
	for _, want := range []string{
		`class="better-friends-grid"`,
		`href="https://alice.example/"`,
		`>Alice</a>`,
		`href="http://bob.example/"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q, got %q", want, out)
		}
	}
}

func TestRenderFenceRejectsInvalidLinks(t *testing.T) {
	out, handled, err := (&plugin{}).RenderFence(fenceLanguage, "- Unsafe: [javascript:alert(1)](javascript:alert(1))\n- Missing: [https://valid.example/]", extension.Context{})
	if err != nil || !handled {
		t.Fatalf("RenderFence = (%q, %v, %v), expected handled empty state", out, handled, err)
	}
	if !strings.Contains(out, "better-friends-empty") || strings.Contains(out, "javascript:") {
		t.Fatalf("expected safe empty state, got %q", out)
	}
}

func TestRenderFenceIgnoresOtherLanguages(t *testing.T) {
	out, handled, err := (&plugin{}).RenderFence("go", "", extension.Context{})
	if err != nil || handled || out != "" {
		t.Fatalf("RenderFence = (%q, %v, %v), want (empty, false, nil)", out, handled, err)
	}
}
