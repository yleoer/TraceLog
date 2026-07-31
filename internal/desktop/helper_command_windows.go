//go:build windows

package desktop

import (
	"os/exec"
	"syscall"
)

func configureUpdateHelperCommand(command *exec.Cmd) {
	const (
		detachedProcess    = 0x00000008
		createNoWindow     = 0x08000000
		createNewProcGroup = 0x00000200
	)
	command.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: detachedProcess | createNoWindow | createNewProcGroup,
		HideWindow:    true,
	}
}
