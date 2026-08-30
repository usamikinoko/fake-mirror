package builder

import (
	"fmt"

	"rainhush/internal/config"
)

func (ctx *buildContext) renderDeep() error {
	if config.Cfg == nil || !config.Cfg.Deep.Enabled {
		return nil
	}
	fmt.Println("Warning: deep.enabled is ignored for static builds; use server-side access control for private content")
	return nil
}
