//go:build windows

package zlog

import "syscall"

var (
	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	procSetConsoleOutputCP = kernel32.NewProc("SetConsoleOutputCP")
)

func init() {
	// Set console output code page to UTF-8 (65001) so non-ASCII characters
	// (e.g. Korean PostgreSQL errors) render correctly on Windows.
	procSetConsoleOutputCP.Call(65001) //nolint:errcheck
}
