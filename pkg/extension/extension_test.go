package extension

import (
	"errors"
	"reflect"
	"testing"
)

type testExtension struct {
	name    string
	matches bool
	output  string
	css     [][]byte
	js      [][]byte
	err     error
}

func (e testExtension) Name() string { return e.name }
func (e testExtension) RenderFence(string, string, Context) (string, bool, error) {
	return e.output, e.matches, e.err
}
func (e testExtension) CSS() [][]byte { return e.css }
func (e testExtension) JS() [][]byte  { return e.js }

func TestRegistryOrdersExtensionsAndAssets(t *testing.T) {
	r := NewRegistry()
	r.Register(testExtension{name: "later", css: [][]byte{[]byte("later")}}, 200)
	r.Register(testExtension{name: "first", css: [][]byte{[]byte("first")}}, 100)

	if got, want := r.Names(), []string{"first", "later"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Names() = %v, want %v", got, want)
	}
	if got, want := string(r.CSS()[0])+","+string(r.CSS()[1]), "first,later"; got != want {
		t.Fatalf("CSS order = %q, want %q", got, want)
	}
}

func TestRegistryDisabledExtensionContributesNothing(t *testing.T) {
	r := NewRegistry()
	r.Register(testExtension{name: "disabled", matches: true, output: "handled", css: [][]byte{[]byte("css")}, js: [][]byte{[]byte("js")}}, 100)
	r.SetEnabled("disabled", false)

	if out, handled := r.RenderFence("demo", "", Context{}); handled || out != "" {
		t.Fatalf("RenderFence() = (%q, %v), want (empty, false)", out, handled)
	}
	if len(r.CSS()) != 0 || len(r.JS()) != 0 {
		t.Fatal("disabled extension must not contribute assets")
	}
}

func TestRegistryUsesFirstHandlingFence(t *testing.T) {
	r := NewRegistry()
	r.Register(testExtension{name: "first", matches: true, output: "one"}, 100)
	r.Register(testExtension{name: "second", matches: true, output: "two"}, 200)

	out, handled := r.RenderFence("demo", "body", Context{Document: "post.md"})
	if !handled || out != "one" {
		t.Fatalf("RenderFence() = (%q, %v), want (one, true)", out, handled)
	}
}

func TestRegistryFenceErrorProducesSafePlaceholder(t *testing.T) {
	r := NewRegistry()
	r.Register(testExtension{name: "broken", matches: true, err: errors.New("<bad>")}, 100)

	out, handled := r.RenderFence("bad", "", Context{})
	if !handled || out == "" || string(out) == "<bad>" {
		t.Fatalf("RenderFence() = (%q, %v), expected safe placeholder", out, handled)
	}
}

func TestRegistryRejectsDuplicateNames(t *testing.T) {
	r := NewRegistry()
	r.Register(testExtension{name: "same"}, 100)
	defer func() {
		if recover() == nil {
			t.Fatal("expected duplicate registration panic")
		}
	}()
	r.Register(testExtension{name: "same"}, 200)
}
