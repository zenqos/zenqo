//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func startCmd(target string) *exec.Cmd {
	c := exec.Command("go", "run", target)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	// CREATE_NEW_PROCESS_GROUP lets us send Ctrl+Break to the child tree.
	c.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
	if err := c.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "  ❌ failed to start: %v\n", err)
		return nil
	}
	return c
}

func stopCmd(c *exec.Cmd) {
	if c == nil || c.Process == nil {
		return
	}
	// Kill the entire process tree (/T) forcefully (/F).
	exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprint(c.Process.Pid)).Run() //nolint:errcheck
	done := make(chan struct{})
	go func() {
		c.Wait() //nolint:errcheck
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		c.Process.Kill() //nolint:errcheck
		<-done
	}
}
