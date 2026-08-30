// Package rainhush provides the public entrypoints for custom Rainhush binaries.
package rainhush

import (
	"rainhush/internal/builder"
	"rainhush/internal/config"
)

// LoadConfig reads _config.yaml from the current working directory.
func LoadConfig() error { return config.Load() }

// Build renders the loaded site configuration into public/.
func Build() error { return builder.Build() }

// BuildSite loads _config.yaml and renders the site into public/.
func BuildSite() error {
	if err := LoadConfig(); err != nil {
		return err
	}
	return Build()
}
