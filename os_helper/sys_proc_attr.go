// +build !linux

package os_helper

import "os/exec"

func ModifyCmdAttributes(cmd *exec.Cmd) *exec.Cmd {
	return cmd
}
