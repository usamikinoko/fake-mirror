package init

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed all:scaffold
var scaffoldFS embed.FS

func Scaffold(targetDir string) error {
	if targetDir == "." {
		var err error
		targetDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("get current directory: %w", err)
		}
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("create directory %s: %w", targetDir, err)
	}

	err := fs.WalkDir(scaffoldFS, "scaffold", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel("scaffold", path)
		if err != nil || relPath == "." {
			return nil
		}
		dest := filepath.Join(targetDir, relPath)
		if d.IsDir() {
			return os.MkdirAll(dest, 0755)
		}
		if _, err := os.Stat(dest); err == nil {
			return fmt.Errorf("refusing to overwrite existing file %s", dest)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("stat %s: %w", dest, err)
		}
		data, err := scaffoldFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read embedded %s: %w", path, err)
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}
		return os.WriteFile(dest, data, 0644)
	})
	if err != nil {
		return fmt.Errorf("scaffold: %w", err)
	}

	fmt.Printf("Initialized rainhush site in %s\n", targetDir)
	return nil
}
