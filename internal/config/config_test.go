package config

import "testing"

func TestExtensionEnabledDefaultsToEnabled(t *testing.T) {
	old := Cfg
	defer func() { Cfg = old }()
	Cfg = &Config{}
	if !ExtensionEnabled("better-friends") {
		t.Fatal("expected omitted extension to be enabled")
	}
}

func TestExtensionEnabledHonorsExplicitFalse(t *testing.T) {
	old := Cfg
	defer func() { Cfg = old }()
	Cfg = &Config{Extensions: map[string]bool{"better-friends": false, "other": true}}
	if ExtensionEnabled("better-friends") {
		t.Fatal("expected explicitly disabled extension to be disabled")
	}
	if !ExtensionEnabled("other") || !ExtensionEnabled("unknown") {
		t.Fatal("expected enabled and unknown extensions to be enabled")
	}
}
