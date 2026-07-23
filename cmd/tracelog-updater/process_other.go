//go:build !windows

package main

func processRunning(pid int) (bool, error) {
	return false, nil
}
