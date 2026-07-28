package main

import (
	"context"
	"fmt"
	"os"

	"rainhush/internal/builder"
	"rainhush/internal/config"
	initpkg "rainhush/internal/init"
	"rainhush/internal/pusher"
	"rainhush/internal/server"
	"rainhush/internal/watcher"
)

var (
	version = "dev"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "--version", "-v":
		fmt.Printf("rainhush version %s\n", version)
		return
	case "--help", "-h", "help":
		printUsage()
		return
	case "init":
		target := "."
		if len(os.Args) > 2 {
			target = os.Args[2]
		}
		if err := initpkg.Scaffold(target); err != nil {
			fmt.Fprintf(os.Stderr, "Init failed: %v\n", err)
			os.Exit(1)
		}
		return
	case "clean", "clear":
		if err := os.RemoveAll("public"); err != nil {
			fmt.Fprintf(os.Stderr, "Clean failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("public/ directory removed.")
		return
	}

	if err := config.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "build":
		if err := builder.Build(); err != nil {
			fmt.Fprintf(os.Stderr, "Build failed: %v\n", err)
			os.Exit(1)
		}
	case "test":
		if err := builder.Build(); err != nil {
			fmt.Fprintf(os.Stderr, "Build failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Watching for changes in content/, templates/, static/...")

		stopWatcher, err := watcher.Watch(func(name string) {
			fmt.Printf("Change detected: %s\n", name)
			if err := builder.Build(); err != nil {
				fmt.Fprintf(os.Stderr, "Rebuild failed: %v\n", err)
			} else {
				fmt.Println("Site rebuilt successfully")
			}
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Watcher failed: %v\n", err)
			os.Exit(1)
		}
		defer stopWatcher()

		if err := server.Serve(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "Serve failed: %v\n", err)
			os.Exit(1)
		}
	case "push":
		if err := builder.Build(); err != nil {
			fmt.Fprintf(os.Stderr, "Build failed: %v\n", err)
			os.Exit(1)
		}
		if err := pusher.Push(); err != nil {
			fmt.Fprintf(os.Stderr, "Push failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Push completed.")
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Rainhush - Static Site Generator")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  rainhush help                  Show this help")
	fmt.Println("  rainhush init [dir]            Create a new site (default: current directory)")
	fmt.Println("  rainhush build                 Build the site from markdown files")
	fmt.Println("  rainhush test                  Build, serve locally, and rebuild on file changes")
	fmt.Println("  rainhush push                  Build and push to remote repository")
	fmt.Println("  rainhush clean                 Remove the public/ build directory")
	fmt.Println("  rainhush clear                 Alias of clean")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -v, --version  Print version")
	fmt.Println("  -h, --help     Print help")
}
