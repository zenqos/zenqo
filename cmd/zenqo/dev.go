package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var defaultEntrypoints = []string{
	"cmd/api/main.go",
	"cmd/server/main.go",
	"main.go",
}

func runDev(args []string) error {
	watch := false
	for _, a := range args {
		switch a {
		case "--watch", "-w":
			watch = true
		case "--help", "-h":
			printDevUsage()
			return nil
		}
	}

	entry, err := detectEntrypoint()
	if err != nil {
		return err
	}

	dir := filepath.Dir(entry)
	if dir == "." {
		dir = ""
	}

	fmt.Printf("\n  ⚡ zenqo dev\n")
	fmt.Printf("     entry: %s\n", entry)
	if watch {
		fmt.Printf("     watch: enabled\n")
	}
	fmt.Println()

	if watch {
		return runWithWatch(dir, entry)
	}
	return runOnce(dir, entry)
}

func printDevUsage() {
	fmt.Print(`
  Usage: zenqo dev [options]

  Options:
    --watch, -w    Watch .go files and restart on changes

  Entrypoint detection order:
    1. cmd/api/main.go
    2. cmd/server/main.go
    3. cmd/*/main.go (auto-select if only one match)
    4. main.go

  Examples:
    zenqo dev
    zenqo dev --watch

`)
}

func detectEntrypoint() (string, error) {
	for _, e := range defaultEntrypoints {
		if _, err := os.Stat(e); err == nil {
			return e, nil
		}
	}

	matches, _ := filepath.Glob("cmd/*/main.go")
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		names := make([]string, len(matches))
		for i, m := range matches {
			names[i] = filepath.Dir(m)
		}
		return "", fmt.Errorf("multiple entrypoints found: %s\nMove your main.go to cmd/api/ or cmd/server/", strings.Join(names, ", "))
	}

	return "", fmt.Errorf("no entrypoint found\nExpected one of: %s", strings.Join(defaultEntrypoints, ", "))
}

func runOnce(dir, entry string) error {
	target := "./" + filepath.Dir(entry)
	if dir == "" {
		target = "."
	}

	cmd := exec.Command("go", "run", target)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func runWithWatch(dir, entry string) error {
	target := "./" + filepath.Dir(entry)
	if dir == "" {
		target = "."
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	var cmd = startCmd(target)

	snapshot := collectGoFileModTimes(".")

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-quit:
			stopCmd(cmd)
			fmt.Println("\n  👋 stopped")
			return nil
		case <-ticker.C:
			current := collectGoFileModTimes(".")
			if !modTimesEqual(snapshot, current) {
				snapshot = current
				fmt.Println("\n  🔄 change detected, restarting...")
				stopCmd(cmd)
				cmd = startCmd(target)
			}
		}
	}
}

func collectGoFileModTimes(root string) map[string]time.Time {
	result := make(map[string]time.Time)
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error { //nolint:errcheck
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if name == "vendor" || name == "node_modules" || name == ".git" || name == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			result[path] = info.ModTime()
		}
		return nil
	})
	return result
}

func modTimesEqual(a, b map[string]time.Time) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || !v.Equal(bv) {
			return false
		}
	}
	return true
}
