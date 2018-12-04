// +build linux

package os_helper

import (
	"os/exec"
	"syscall"
)

func ModifyCmdAttributes(cmd *exec.Cmd) *exec.Cmd {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGTERM,
	}
	return cmd
}
