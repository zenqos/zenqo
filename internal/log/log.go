package zlog

import (
	"fmt"
	"os"
	"time"
)

// isTTY reports whether the given file is connected to a terminal.
func isTTY(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

var (
	colorOut = isTTY(os.Stdout)
	colorErr = isTTY(os.Stderr)
)

// Log prints a structured INFO log in Zenqo format.
// ANSI colors are automatically disabled when stdout is not a TTY.
func Log(label, msg string) {
	ts := time.Now().Format("2006/01/02 15:04:05")
	if colorOut {
		fmt.Fprintf(os.Stdout, "[Zenqo] %s  \033[32mLOG\033[0m  [%s]  %s\n", ts, label, msg)
	} else {
		fmt.Fprintf(os.Stdout, "[Zenqo] %s  LOG  [%s]  %s\n", ts, label, msg)
	}
}

// Warn prints a structured WARN log in Zenqo format.
// ANSI colors are automatically disabled when stderr is not a TTY.
func Warn(label, msg string) {
	ts := time.Now().Format("2006/01/02 15:04:05")
	if colorErr {
		fmt.Fprintf(os.Stderr, "[Zenqo] %s  \033[33mWARN\033[0m [%s]  %s\n", ts, label, msg)
	} else {
		fmt.Fprintf(os.Stderr, "[Zenqo] %s  WARN [%s]  %s\n", ts, label, msg)
	}
}

// Err prints a structured ERROR log in Zenqo format.
// ANSI colors are automatically disabled when stderr is not a TTY.
func Err(label, msg string) {
	ts := time.Now().Format("2006/01/02 15:04:05")
	if colorErr {
		fmt.Fprintf(os.Stderr, "[Zenqo] %s  \033[31mERR\033[0m  [%s]  %s\n", ts, label, msg)
	} else {
		fmt.Fprintf(os.Stderr, "[Zenqo] %s  ERR  [%s]  %s\n", ts, label, msg)
	}
}
