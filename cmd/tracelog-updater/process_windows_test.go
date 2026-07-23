//go:build windows

package main

import (
	"os"
	"testing"
)

func TestProcessRunning(t *testing.T) {
	running, err := processRunning(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if !running {
		t.Fatal("expected the current process to be running")
	}

	running, err = processRunning(0x7fffffff)
	if err != nil {
		t.Fatal(err)
	}
	if running {
		t.Fatal("expected an unused process ID to be stopped")
	}
}
