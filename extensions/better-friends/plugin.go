// Package betterfriends renders friend-link fenced blocks as accessible card grids.
package betterfriends

import (
	_ "embed"
	"html"
	"net/url"
	"strings"

	"rainhush/pkg/extension"
)

//go:embed assets/better-friends.css
var cssAsset []byte

const fenceLanguage = "better-friends"

type plugin struct{}

type friend struct {
	name string
	url  string
}

var _ extension.Fence = (*plugin)(nil)
var _ extension.Assets = (*plugin)(nil)

func init() { extension.Register(&plugin{}, 200) }

func (*plugin) Name() string  { return "better-friends" }
func (*plugin) CSS() [][]byte { return [][]byte{cssAsset} }
func (*plugin) JS() [][]byte  { return nil }

// RenderFence renders lines in the form "- Name: [https://example.com/](https://example.com/)".
func (*plugin) RenderFence(lang, content string, _ extension.Context) (string, bool, error) {
	if lang != fenceLanguage {
		return "", false, nil
	}

	friends := parseFriends(content)
	if len(friends) == 0 {
		return `<p class="better-friends-empty">No valid friend links were found.</p>`, true, nil
	}

	var out strings.Builder
	out.WriteString(`<nav class="better-friends-grid" aria-label="Friend links">`)
	for _, item := range friends {
		out.WriteString(`<a class="better-friends-card" href="`)
		out.WriteString(html.EscapeString(item.url))
		out.WriteString(`" rel="noopener noreferrer">`)
		out.WriteString(html.EscapeString(item.name))
		out.WriteString(`</a>`)
	}
	out.WriteString(`</nav>`)
	return out.String(), true, nil
}

func parseFriends(content string) []friend {
	var friends []friend
	for _, line := range strings.Split(content, "\n") {
		name, target, ok := parseFriendLine(strings.TrimSpace(line))
		if ok {
			friends = append(friends, friend{name: name, url: target})
		}
	}
	return friends
}

func parseFriendLine(line string) (name, target string, ok bool) {
	line = strings.TrimSpace(strings.TrimPrefix(line, "-"))
	name, value, found := strings.Cut(line, ":")
	if !found || strings.TrimSpace(name) == "" {
		return "", "", false
	}

	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "[") {
		return "", "", false
	}
	closeLabel := strings.Index(value, "](")
	if closeLabel < 1 || !strings.HasSuffix(value, ")") {
		return "", "", false
	}
	label := strings.TrimSpace(value[1:closeLabel])
	target = strings.TrimSpace(value[closeLabel+2 : len(value)-1])
	if label == "" || !safeURL(target) {
		return "", "", false
	}
	return strings.TrimSpace(name), target, true
}

func safeURL(raw string) bool {
	parsed, err := url.Parse(raw)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}
