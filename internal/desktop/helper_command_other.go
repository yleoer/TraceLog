//go:build !windows

package desktop

import "os/exec"

func configureUpdateHelperCommand(_ *exec.Cmd) {}
